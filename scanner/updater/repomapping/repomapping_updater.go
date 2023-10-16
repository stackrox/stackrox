package repomapping

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/scanner/updater"
)

const (
	repoMappingFilename = "rhelv2/repo.zip"
	defaultTimeout      = 5 * time.Minute
	defaultInterval     = 4 * time.Hour
)

// RepoMappingUpdater updates the Scanner's container-name-repos-map.json and repository-to-cpe.json for scanner indexer, contacting
// Sensor, instead of Central.
type RepoMappingUpdater struct {
	interval        time.Duration
	lastUpdatedTime time.Time
	stopSig         *concurrency.Signal

	sensorClient             *http.Client
	repoMappingUrl           string
	repoMappingLocalFilename string
}

// NewRepoMappingUpdater creates and initialize a new repomapping updater.
func NewRepoMappingUpdater(sensorEndpoint string) (*RepoMappingUpdater, error) {
	repoToCPEURL, err := urlfmt.FullyQualifiedURL(
		strings.Join([]string{
			urlfmt.FormatURL(sensorEndpoint, urlfmt.HTTPS, urlfmt.NoTrailingSlash),
			"scanner-v4/repomappings",
		}, "/"),
		url.Values{})
	if err != nil {
		return nil, errors.Wrapf(err, "setting up sensor URL at %s", sensorEndpoint)
	}

	// Create sensor's HTTP client.
	sensorClient, err := clientconn.NewHTTPClient(
		mtls.SensorSubject, urlfmt.FormatURL(sensorEndpoint, urlfmt.NONE, urlfmt.NoTrailingSlash), defaultTimeout)
	if err != nil {
		return nil, errors.Wrap(err, "creating sensor client")
	}

	// Set up the repo2cpe local filename and its directory.
	repoToCPELocalFilename := filepath.Join(slimUpdaterDir, filepath.FromSlash(repoMappingFilename))
	if err := os.MkdirAll(filepath.Dir(repoToCPELocalFilename), 0700); err != nil {
		return nil, errors.Wrap(err, "creating slim updater output dir")
	}

	// Initialize the updater object.
	stopSig := concurrency.NewSignal()
	updater := &RepoMappingUpdater{
		interval:                 defaultInterval,
		stopSig:                  &stopSig,
		sensorClient:             sensorClient,
		repoMappingLocalFilename: repoToCPELocalFilename,
		repoMappingUrl:           repoToCPEURL,
	}

	return updater, nil
}

// RunForever starts the updater loop.
func (u *RepoMappingUpdater) RunForever() {
	t := time.NewTicker(u.interval)
	defer t.Stop()
	for {
		if err := u.update(); err != nil {
			logrus.WithError(err).Error("repo mapping data update failed")
		}
		select {
		case <-t.C:
			continue
		case <-u.stopSig.Done():
			return
		}
	}

}

// Stop stops the updater loop.
func (u *RepoMappingUpdater) Stop() {
	u.stopSig.Signal()
}

// update performs the slim updater steps.
func (u *RepoMappingUpdater) update() error {
	logrus.Info("starting slim update")
	startTime := time.Now()
	fetched, err := updater.FetchDumpFromURL(
		u.stopSig,
		u.sensorClient,
		u.repoMappingUrl,
		u.lastUpdatedTime,
		u.repoMappingLocalFilename,
	)
	if err != nil {
		return errors.Wrap(err, "fetching update from URL")
	}
	if !fetched {
		logrus.Info("already up-to-date, nothing to do")
		return nil
	}
	u.lastUpdatedTime = startTime
	logrus.Info("Finished repo mapping data update.")
	return nil
}
