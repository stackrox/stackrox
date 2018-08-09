package detection

import (
	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/enrichment"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	policyDataStore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/compiledpolicies"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	logger = logging.LoggerForModule()
)

// Detector processes deployments and reports alerts if policies are violated.
type Detector interface {
	Stop()

	EnrichAndReprocess()

	Detect(task Task) (alert *v1.Alert, enforcement v1.EnforcementAction, excluded *v1.DryRunResponse_Excluded)

	ProcessDeploymentEvent(deployment *v1.Deployment, action v1.ResourceAction) (alertID string, enforcement v1.EnforcementAction)

	UpdatePolicy(policy compiledpolicies.DeploymentMatcher)
	RemovePolicy(id string)

	RemoveNotifier(id string)
}

// New creates a new detector and initializes the registries and scanners from the DB if they exist.
func New(alertStorage alertDataStore.DataStore,
	deploymentStorage deploymentDataStore.DataStore,
	policyStorage policyDataStore.DataStore,
	imageStorage imageDataStore.DataStore,
	enricher enrichment.Enricher,
	notificationsProcessor notifierProcessor.Processor) (Detector, error) {
	d := &detectorImpl{
		alertStorage:          alertStorage,
		deploymentStorage:     deploymentStorage,
		policyStorage:         policyStorage,
		imageStorage:          imageStorage,
		enricher:              enricher,
		notificationProcessor: notificationsProcessor,
		taskC:    make(chan Task, 40),
		stoppedC: make(chan struct{}),
	}

	if err := d.initializePolicies(); err != nil {
		return nil, err
	}

	go d.periodicallyEnrich()
	go d.reprocessLoop()

	return d, nil
}
