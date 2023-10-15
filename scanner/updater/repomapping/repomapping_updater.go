package repomapping

import (
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/urlfmt"
)

// RepoMappingUpdater updates the Scanner's container-name-repos-map.json and repository-to-cpe.json for scanner indexer, contacting
// Sensor, instead of Central.
type RepoMappingUpdater struct {
	interval        time.Duration
	lastUpdatedTime time.Time
	stopSig         *concurrency.Signal

	sensorClient *http.Client
	repoToCPEURL string
}

// NewRepoMappingUpdater creates and initialize a new repomapping updater.
func NewRepoMappingUpdater(updaterConfig Config, sensorEndpoint string) (*RepoMappingUpdater, error) {
	repoToCPEURL, err := urlfmt.FullyQualifiedURL(
		strings.Join([]string{
			urlfmt.FormatURL(sensorEndpoint, urlfmt.HTTPS, urlfmt.NoTrailingSlash),
			"scanner-v4/repomappings",
		}, "/"))
	if err != nil {
		return nil, errors.Wrapf(err, "setting up sensor URL at %s", sensorEndpoint)
	}

	// Create sensor's HTTP client.
	sensorClient, err := clientconn.NewHTTPClient(
		mtls.SensorSubject, urlfmt.FormatURL(sensorEndpoint, urlfmt.NONE, urlfmt.NoTrailingSlash), defaultTimeout)
	if err != nil {
		return nil, errors.Wrap(err, "creating sensor client")
	}

	// Initialize the updater object.
	stopSig := concurrency.NewSignal()
	updater := &RepoMappingUpdater{
		interval:     updaterConfig.Interval,
		stopSig:      &stopSig,
		sensorClient: sensorClient,
		repoToCPEURL: repoToCPEURL,
	}

	return updater, nil
}
