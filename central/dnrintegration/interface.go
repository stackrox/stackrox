package dnrintegration

import (
	"fmt"
	"time"

	"github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	"golang.org/x/time/rate"
)

var (
	logger = logging.LoggerForModule()
)

// DNRIntegration exposes all functionality that we expect to get through the integration with Detect & Respond.
type DNRIntegration interface {
	// Alerts returns D&R alerts for a deployment, given the cluster id, namespace and service name.
	Alerts(clusterID, namespace, serviceName string) (AlertsWithMetadata, error)
}

// New returns a ready-to-use DNRIntegration object from the proto.
func New(integration *v1.DNRIntegration, deploymentDataStore datastore.DataStore) (DNRIntegration, error) {
	portalURL, err := validateAndParsePortalURL(integration.GetPortalUrl())
	if err != nil {
		return nil, fmt.Errorf("portal URL failed validation/parsing: %s", err)
	}

	d := &dnrIntegrationImpl{
		portalURL: portalURL,
		authToken: integration.GetAuthToken(),
		client:    client,

		deploymentStore: deploymentDataStore,
	}

	err = d.initialize(integration.GetClusterIds())
	if err != nil {
		return nil, err
	}
	err = d.refreshServiceMappings()
	if err != nil {
		return nil, err
	}

	d.serviceMappingsRateLimiter = rate.NewLimiter(rate.Every(time.Minute), 2)
	return d, nil
}
