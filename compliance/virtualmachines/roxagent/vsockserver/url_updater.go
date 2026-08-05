package vsockserver

import (
	"context"
	"errors"
	"os"
	"time"

	pb "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/filedownloader"
	"github.com/stackrox/rox/pkg/scannerv4/repositorytocpe"
	"github.com/stackrox/rox/pkg/sync"
)

// Retry/refresh policy for the periodic mapping download, mirroring
// cmd.newMappingDownloader: WaitMin=5s with the default WaitMax=30s covers
// four HTTP attempts (5+10+20=35s) within urlMappingClientTimeout.
const (
	urlMappingFetchRetryMax     = 3
	urlMappingFetchRetryWaitMin = 5 * time.Second
	urlMappingClientTimeout     = 2 * time.Minute
	urlMappingRefreshInterval   = time.Hour
)

var _ MappingProvider = (*URLUpdater)(nil)

var errURLMappingNotReady = errors.New("no repo-to-CPE mapping available yet")

// URLUpdater is the MappingProvider for an agent configured with a
// mapping URL; this mapping source never accepts a Sensor-pushed Update.
type URLUpdater struct {
	downloader *filedownloader.Downloader
	cachePath  string
	active     []byte
	activeHash string
	onChange   func()
	mu         sync.Mutex
}

// NewURLUpdater seeds active from cachePath if it holds a validated
// mapping (e.g. left by a prior process's successful download), else
// stays empty until the first successful fetch. There is no bundled-file
// fallback: a URL-backed agent has exactly one mapping source.
func NewURLUpdater(url, cachePath string, onChange func()) *URLUpdater {
	u := &URLUpdater{cachePath: cachePath, onChange: onChange}
	if u.bootstrap() && onChange != nil {
		onChange()
	}
	u.downloader = filedownloader.New(url, cachePath, urlMappingRefreshInterval,
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
		return false
	}
	if err := repositorytocpe.ValidateMapping(content); err != nil {
		log.Warnf("Ignoring invalid repo-to-CPE mapping cache at %q: %v", u.cachePath, err)
		return false
	}
	u.active = content
	u.activeHash = repositorytocpe.HashMapping(content)
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

// Path returns cachePath, which the downloader already keeps in sync with
// the active mapping via its own atomic writes.
func (u *URLUpdater) Path() (string, error) {
	u.mu.Lock()
	ready := len(u.active) > 0
	u.mu.Unlock()
	if !ready {
		return "", errURLMappingNotReady
	}
	return u.cachePath, nil
}

// onDownloadComplete is filedownloader's OnComplete callback: it logs and
// keeps the last-good mapping on failure, or re-validates before promoting
// on success, so a corrupted file can never replace a good mapping.
func (u *URLUpdater) onDownloadComplete(err error, _ time.Duration) {
	if err != nil {
		log.Warnf("Downloading repo-to-CPE mapping: %v", err)
		return
	}
	content, err := os.ReadFile(u.cachePath)
	if err != nil {
		log.Warnf("Reading downloaded repo-to-CPE mapping at %q: %v", u.cachePath, err)
		return
	}
	if err := repositorytocpe.ValidateMapping(content); err != nil {
		log.Warnf("Downloaded repo-to-CPE mapping failed validation, keeping last-good: %v", err)
		return
	}
	hash := repositorytocpe.HashMapping(content)

	u.mu.Lock()
	unchanged := hash == u.activeHash
	u.active = content
	u.activeHash = hash
	u.mu.Unlock()

	if !unchanged && u.onChange != nil {
		u.onChange()
	}
}
