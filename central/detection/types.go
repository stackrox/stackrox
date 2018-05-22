package detection

import (
	"sync"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/central/detection/matcher"
	"bitbucket.org/stack-rox/apollo/central/enrichment"
	"bitbucket.org/stack-rox/apollo/central/notifications"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/sources"
)

var (
	logger = logging.LoggerForModule()
)

// Detector processes deployments and reports alerts if policies are violated.
type Detector struct {
	alertStorage      db.AlertStorage
	deploymentStorage db.DeploymentStorage
	policyStorage     db.PolicyStorage

	enricher              *enrichment.Enricher
	notificationProcessor *notifications.Processor
	taskC                 chan Task
	stopping              bool
	stoppedC              chan struct{}

	policyMutex sync.RWMutex
	policies    map[string]*matcher.Policy
}

// New creates a new detector and initializes the registries and scanners from the DB if they exist.
func New(alertStorage db.AlertStorage,
	deploymentStorage db.DeploymentStorage,
	policyStorage db.PolicyStorage,
	enricher *enrichment.Enricher,
	notificationsProcessor *notifications.Processor) (d *Detector, err error) {
	d = &Detector{
		alertStorage:          alertStorage,
		deploymentStorage:     deploymentStorage,
		policyStorage:         policyStorage,
		enricher:              enricher,
		notificationProcessor: notificationsProcessor,
		taskC:    make(chan Task, 40),
		stoppedC: make(chan struct{}),
	}

	if err = d.initializePolicies(); err != nil {
		return nil, err
	}

	go d.periodicallyEnrich()
	go d.reprocessLoop()

	return d, nil
}

// Stop closes the Task reprocessing channel, and waits for remaining tasks to finish before returning.
func (d *Detector) Stop() {
	d.stopping = true
	close(d.taskC)
	<-d.stoppedC
}

// NewTask creates a new task object
func NewTask(deployment *v1.Deployment, action v1.ResourceAction, policy *matcher.Policy) Task {
	return Task{
		deployment: deployment,
		action:     action,
		policy:     policy,
	}
}

// Task describes a unit to be processed
type Task struct {
	deployment *v1.Deployment
	action     v1.ResourceAction
	policy     *matcher.Policy
}

// UpdateImageIntegration updates the map of active integrations
func (d *Detector) UpdateImageIntegration(integration *sources.ImageIntegration) {
	d.enricher.UpdateImageIntegration(integration)
	go d.reprocessImageIntegration(integration)
}

// RemoveImageIntegration removes an image integration
func (d *Detector) RemoveImageIntegration(id string) {
	d.enricher.RemoveImageIntegration(id)
}
