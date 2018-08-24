package enforcers

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	logger = logging.LoggerForModule()
)

// CreateEnforcer creates a new enforcer that performs the given enforcement actions.
func CreateEnforcer(enforcementMap map[v1.EnforcementAction]EnforceFunc) Enforcer {
	return &enforcer{
		enforcementMap: enforcementMap,
		actionsC:       make(chan *DeploymentEnforcement, 10),
		stopC:          concurrency.NewSignal(),
		stoppedC:       concurrency.NewSignal(),
	}
}

type enforcer struct {
	enforcementMap map[v1.EnforcementAction]EnforceFunc
	actionsC       chan *DeploymentEnforcement
	stopC          concurrency.Signal
	stoppedC       concurrency.Signal
}

func (e *enforcer) Actions() chan<- *DeploymentEnforcement {
	return e.actionsC
}

func (e *enforcer) Start() {
	for {
		select {
		case action := <-e.actionsC:
			f, ok := e.enforcementMap[action.Enforcement]
			if !ok {
				logger.Errorf("unknown enforcement action: %s", action.Enforcement)
				continue
			}

			if err := f(action); err != nil {
				logger.Errorf("failed to take enforcement action %s on deployment %s: %s", action.Enforcement, action.Deployment.GetName(), err)
			} else {
				logger.Infof("Successfully taken %s on deployment %s", action.Enforcement, action.Deployment.GetName())
			}
		case <-e.stopC.Done():
			logger.Info("Shutting down Enforcer")
			e.stoppedC.Signal()
		}
	}
}

func (e *enforcer) Stop() {
	e.stopC.Signal()
	e.stoppedC.Wait()
}
