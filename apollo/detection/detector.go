package detection

import (
	"sync"

	"bitbucket.org/stack-rox/apollo/apollo/db"
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
		db.PolicyStorage
		db.ImageStorage
		db.RegistryStorage
		db.ScannerStorage
	}

	policyMutex sync.Mutex
	policies    map[string]*policyWrapper

	registryMutex sync.Mutex
	registries    map[string]registries.ImageRegistry

	scannerMutex sync.Mutex
	scanners     map[string]scannerTypes.ImageScanner
}

// New creates a new detector and initializes the registries and scanners from the DB if they exist.
func New(database db.Storage) (*Detector, error) {
	d := &Detector{
		database: database,
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
	return d, nil
}

// Process takes in a deployment and return alerts.
func (d *Detector) Process(deployment *v1.Deployment) (alerts []*v1.Alert, err error) {
	if err = d.enrich(deployment); err != nil {
		return
	}

	d.policyMutex.Lock()
	defer d.policyMutex.Unlock()

	for _, policy := range d.policies {
		if policy.GetDisabled() {
			continue
		}

		if alert := d.matchPolicy(deployment, policy); alert != nil {
			alerts = append(alerts, alert)
		}
	}

	return
}
