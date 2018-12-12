package enforcers

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	logger = logging.LoggerForModule()
)

// EnforceFunc represents an enforcement function.
type EnforceFunc func(*v1.SensorEnforcement) error

// Enforcer is an abstraction for taking enforcement actions on deployments.
type Enforcer interface {
	Actions() chan<- *v1.SensorEnforcement
	Start()
	Stop()
}

// CreateEnforcer creates a new enforcer that performs the given enforcement actions.
func CreateEnforcer(enforcementMap map[storage.EnforcementAction]EnforceFunc) Enforcer {
	return &enforcer{
		enforcementMap: enforcementMap,
		actionsC:       make(chan *v1.SensorEnforcement, 10),
		stopC:          concurrency.NewSignal(),
		stoppedC:       concurrency.NewSignal(),
	}
}

type enforcer struct {
	enforcementMap map[storage.EnforcementAction]EnforceFunc
	actionsC       chan *v1.SensorEnforcement
	stopC          concurrency.Signal
	stoppedC       concurrency.Signal
}

func (e *enforcer) Actions() chan<- *v1.SensorEnforcement {
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
				logger.Errorf("failed to take enforcement action: %s err: %s", proto.MarshalTextString(action), err)
			} else {
				logger.Infof("Successfully taken action %s", proto.MarshalTextString(action))
			}
		case <-e.stopC.Done():
			logger.Info("Shutting down Enforcer")
			e.stoppedC.Signal()
			return
		}
	}
}

func (e *enforcer) Stop() {
	e.stopC.Signal()
	e.stoppedC.Wait()
}
