package enforcers

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	logger = logging.LoggerForModule()
)

// EnforceFunc represents an enforcement function.
type EnforceFunc func(*central.SensorEnforcement) error

// Enforcer is an abstraction for taking enforcement actions on deployments.
type Enforcer interface {
	SendEnforcement(*central.SensorEnforcement) bool
	Start()
	Stop()
}

// CreateEnforcer creates a new enforcer that performs the given enforcement actions.
func CreateEnforcer(enforcementMap map[storage.EnforcementAction]EnforceFunc) Enforcer {
	return &enforcer{
		enforcementMap: enforcementMap,
		actionsC:       make(chan *central.SensorEnforcement, 10),
		stopC:          concurrency.NewSignal(),
		stoppedC:       concurrency.NewSignal(),
	}
}

type enforcer struct {
	enforcementMap map[storage.EnforcementAction]EnforceFunc
	actionsC       chan *central.SensorEnforcement
	stopC          concurrency.Signal
	stoppedC       concurrency.Signal
}

func (e *enforcer) SendEnforcement(enforcement *central.SensorEnforcement) bool {
	select {
	case e.actionsC <- enforcement:
		return true
	case <-e.stoppedC.Done():
		return false
	}
}

func (e *enforcer) Start() {
	defer e.stoppedC.Signal()

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
			return
		}
	}
}

func (e *enforcer) Stop() {
	e.stopC.Signal()
	e.stoppedC.Wait()
}
