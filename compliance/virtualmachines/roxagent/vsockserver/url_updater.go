package vsockserver

import (
	"context"
	"errors"
	"os"
	"time"

	pb "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/filedownloader"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/virtualmachine/cpemapping"
)

// Retry/refresh policy for the periodic mapping download. WaitMin=5s with
// retryablehttp's default WaitMax=30s covers four HTTP attempts (5+10+20=35s)
// within urlMappingClientTimeout.
const (
	urlMappingFetchRetryMax     = 3
	urlMappingFetchRetryWaitMin = 5 * time.Second
	urlMappingClientTimeout     = 2 * time.Minute
	urlMappingRefreshInterval   = time.Hour
	// urlMappingStagingSuffix names the path the downloader writes to.
	// onDownloadComplete only promotes a staged download to cachePath
	// once it passes validation, so a bad fetch can never corrupt the
	// persisted last-good mapping.
	urlMappingStagingSuffix = ".download"
)

var _ MappingProvider = (*URLUpdater)(nil)

var errURLMappingNotReady = errors.New("no repo-to-CPE mapping available yet")

// URLUpdater is the MappingProvider for an agent configured with a
// mapping URL; this mapping source never accepts a Sensor-pushed Update.
type URLUpdater struct {
	downloader  *filedownloader.Downloader
	cachePath   string
	stagingPath string
	active      []byte
	activeHash  string
	onChange    func()
	mu          sync.Mutex
}

// NewURLUpdater seeds active from cachePath (last-good from a prior
// successful download), else stays empty until the first fetch. Bundled
// image content is not used: a never-working URL would look configured.
func NewURLUpdater(url, cachePath string, onChange func()) *URLUpdater {
	u := &URLUpdater{cachePath: cachePath, stagingPath: cachePath + urlMappingStagingSuffix, onChange: onChange}
	if u.bootstrap() && onChange != nil {
		onChange()
	}
	u.downloader = filedownloader.New(url, u.stagingPath, urlMappingRefreshInterval,
		filedownloader.WithRequestTimeout(urlMappingClientTimeout),
		filedownloader.WithRetryMax(urlMappingFetchRetryMax),
		filedownloader.WithRetryWaitMin(urlMappingFetchRetryWaitMin),
		filedownloader.WithOnComplete(u.onDownloadComplete),
	)
	return u
}

// bootstrap seeds active/activeHash from cachePath if it decodes as a
// valid mapping. Reports whether it seeded active.
func (u *URLUpdater) bootstrap() bool {
	content, err := os.ReadFile(u.cachePath)
	if err != nil {
		// First boot always misses cache; do not treat that as an error.
		if !errors.Is(err, os.ErrNotExist) {
			log.Warnf("Reading repo-to-CPE mapping cache at %q: %v", u.cachePath, err)
		}
		return false
	}
	if err := cpemapping.ValidateMapping(content); err != nil {
		log.Warnf("Ignoring invalid repo-to-CPE mapping cache at %q: %v", u.cachePath, err)
		return false
	}
	u.active = content
	u.activeHash = cpemapping.HashMapping(content)
	return true
}

// Start begins the periodic download loop in the background; agent
// startup never blocks on a successful fetch.
func (u *URLUpdater) Start(ctx context.Context) error {
	return u.downloader.Start(ctx, false)
}

// Stop signals the download loop to stop and blocks until it exits.
func (u *URLUpdater) Stop() {
	u.downloader.Stop()
}

// Ready reports whether a validated mapping is currently active.
func (u *URLUpdater) Ready() bool {
	u.mu.Lock()
	defer u.mu.Unlock()
	return len(u.active) > 0
}

// Hash returns the active mapping's content hash, or "" if not Ready.
func (u *URLUpdater) Hash() string {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.activeHash
}

// UpdatePath reports this updater's mapping source for ResponseMeta.
func (u *URLUpdater) UpdatePath() pb.RepoCPEMappingUpdatePath {
	return pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_URL
}

// Bytes returns a copy of the active mapping content.
func (u *URLUpdater) Bytes() ([]byte, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if len(u.active) == 0 {
		return nil, errURLMappingNotReady
	}
	out := make([]byte, len(u.active))
	copy(out, u.active)
	return out, nil
}

// Path returns cachePath. onDownloadComplete only sets active after a
// validated download has been persisted there, so the file matches Bytes().
func (u *URLUpdater) Path() (string, error) {
	ready := concurrency.WithLock1(&u.mu, func() bool {
		return len(u.active) > 0
	})
	if !ready {
		return "", errURLMappingNotReady
	}
	return u.cachePath, nil
}

// onDownloadComplete applies a staged download only after it validates
// and persists to cachePath, so Bytes() and Path() stay aligned and a
// failed persist is retried on the next identical fetch.
func (u *URLUpdater) onDownloadComplete(err error, _ time.Duration) {
	if err != nil {
		log.Warnf("Error downloading repo-to-CPE mapping: %v", err)
		return
	}
	content, err := os.ReadFile(u.stagingPath)
	if err != nil {
		log.Warnf("Reading downloaded repo-to-CPE mapping at %q: %v", u.stagingPath, err)
		return
	}
	if err := cpemapping.ValidateMapping(content); err != nil {
		log.Warnf("Downloaded repo-to-CPE mapping failed validation, keeping last-good: %v", err)
		return
	}
	hash := cpemapping.HashMapping(content)

	alreadyActive := concurrency.WithLock1(&u.mu, func() bool {
		return hash == u.activeHash
	})
	if alreadyActive {
		return
	}
	if err := filedownloader.AtomicWriteFile(u.cachePath, content); err != nil {
		log.Warnf("Persisting repo-to-CPE mapping cache to %q: %v", u.cachePath, err)
		return
	}
	concurrency.WithLock(&u.mu, func() {
		u.active = content
		u.activeHash = hash
	})
	if u.onChange != nil {
		u.onChange()
	}
}
