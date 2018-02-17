package detection

import (
	"sync"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/central/detection/matcher"
	"bitbucket.org/stack-rox/apollo/central/enrichment"
	"bitbucket.org/stack-rox/apollo/central/notifications"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/registries"
	scannerTypes "bitbucket.org/stack-rox/apollo/pkg/scanners"
)

var (
	logger = logging.LoggerForModule()
)

// Detector processes deployments and reports alerts if policies are violated.
type Detector struct {
	database interface {
		db.AlertStorage
		db.DeploymentStorage
		db.PolicyStorage
	}

	enricher              *enrichment.Enricher
	notificationProcessor *notifications.Processor
	taskC                 chan Task
	stopping              bool
	stoppedC              chan struct{}

	policyMutex sync.Mutex
	policies    map[string]*matcher.Policy
}

// New creates a new detector and initializes the registries and scanners from the DB if they exist.
func New(database db.Storage, notificationsProcessor *notifications.Processor) (d *Detector, err error) {
	d = &Detector{
		database:              database,
		notificationProcessor: notificationsProcessor,
		taskC:    make(chan Task, 40),
		stoppedC: make(chan struct{}),
	}

	if d.enricher, err = enrichment.New(database); err != nil {
		return nil, err
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

// UpdateRegistry updates image processors map of active registries
func (d *Detector) UpdateRegistry(registry registries.ImageRegistry) {
	d.enricher.UpdateRegistry(registry)
	go d.reprocessRegistry(registry)
}

// RemoveRegistry removes a registry from image processors map of active registries
func (d *Detector) RemoveRegistry(id string) {
	d.enricher.RemoveRegistry(id)
}

// UpdateScanner updates image processors map of active scanners
func (d *Detector) UpdateScanner(scanner scannerTypes.ImageScanner) {
	d.enricher.UpdateScanner(scanner)
	go d.reprocessScanner(scanner)
}

// RemoveScanner removes a scanner from image processors map of active scanners
func (d *Detector) RemoveScanner(id string) {
	d.enricher.RemoveScanner(id)
}
