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
func (d *Detector) Process(deployment *v1.Deployment, action v1.ResourceAction) (alerts []*v1.Alert, enforcement v1.EnforcementAction, err error) {
	if err = d.enrich(deployment); err != nil {
		return
	}

	var enforcementActions []v1.EnforcementAction

	d.policyMutex.Lock()
	defer d.policyMutex.Unlock()

	for _, policy := range d.policies {
		if !policy.shouldProcess(deployment) {
			continue
		}

		if alert := d.matchPolicy(deployment, policy); alert != nil {
			if action, msg := policy.getEnforcementAction(deployment, action); action != v1.EnforcementAction_UNSET_ENFORCEMENT {
				enforcementActions = append(enforcementActions, action)
				alert.Enforcement = &v1.Alert_Enforcement{
					Action:  action,
					Message: msg,
				}
			}

			alerts = append(alerts, alert)
		}
	}

	enforcement = d.determineEnforcementResponse(enforcementActions)
	return
}

// Each alert can have an enforcement response, but (assuming that enforcement is mutually exclusive) only one can be
// taken per deployment.
// Currently a Scale to 0 enforcement response is issued if any alert raises this action.
func (d *Detector) determineEnforcementResponse(enforcementActions []v1.EnforcementAction) v1.EnforcementAction {
	for _, a := range enforcementActions {
		if a == v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT {
			return v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT
		}
	}

	return v1.EnforcementAction_UNSET_ENFORCEMENT
}
