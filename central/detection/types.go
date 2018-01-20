package detection

import (
	"sync"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/central/detection/matcher"
	"bitbucket.org/stack-rox/apollo/central/notifications"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/registries"
	scannerTypes "bitbucket.org/stack-rox/apollo/pkg/scanners"
)

var (
	logger = logging.New("detection")
)

// Detector processes deployments and reports alerts if policies are violated.
type Detector struct {
	database interface {
		db.AlertStorage
		db.DeploymentStorage
		db.ImageStorage
		db.PolicyStorage
		db.RegistryStorage
		db.ScannerStorage
	}

	notificationProcessor *notifications.Processor
	taskC                 chan task
	stopping              bool
	stoppedC              chan struct{}

	policyMutex sync.Mutex
	policies    map[string]*matcher.Policy

	registryMutex sync.Mutex
	registries    map[string]registries.ImageRegistry

	scannerMutex sync.Mutex
	scanners     map[string]scannerTypes.ImageScanner
}

// New creates a new detector and initializes the registries and scanners from the DB if they exist.
func New(database db.Storage, notificationsProcessor *notifications.Processor) (*Detector, error) {
	d := &Detector{
		database:              database,
		notificationProcessor: notificationsProcessor,
		taskC:    make(chan task, 40),
		stoppedC: make(chan struct{}),
	}

	if err := d.initializePolicies(); err != nil {
		return nil, err
	}
	if err := d.initializeRegistries(); err != nil {
		return nil, err
	}
	if err := d.initializeScanners(); err != nil {
		return nil, err
	}

	go d.periodicallyEnrich()
	go d.reprocessLoop()

	return d, nil
}

// Stop closes the task reprocessing channel, and waits for remaining tasks to finish before returning.
func (d *Detector) Stop() {
	d.stopping = true
	close(d.taskC)
	<-d.stoppedC
}

type task struct {
	deployment *v1.Deployment
	action     v1.ResourceAction
	policy     *matcher.Policy
}
