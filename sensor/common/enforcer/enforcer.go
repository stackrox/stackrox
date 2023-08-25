package enforcer

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/enforcers"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
)

var (
	log = logging.LoggerForModule()
)

// Enforcer implements the interface to apply enforcement to a sensor cluster
type Enforcer interface {
	common.SensorComponent
	ProcessAlertResults(action central.ResourceAction, stage storage.LifecycleStage, alertResults *central.AlertResults)
}

// EnforceFunc represents an enforcement function.
type EnforceFunc func(context.Context, *central.SensorEnforcement) error

// CreateEnforcer creates a new enforcer that performs the given enforcement actions.
func CreateEnforcer(enforcementMap map[storage.EnforcementAction]EnforceFunc) Enforcer {
	return &enforcer{
		enforcementMap: enforcementMap,
		actionsC:       make(chan *central.SensorEnforcement, 10),
		stopper:        concurrency.NewStopper(),
	}
}

type enforcer struct {
	enforcementMap map[storage.EnforcementAction]EnforceFunc
	actionsC       chan *central.SensorEnforcement
	stopper        concurrency.Stopper
}

func (e *enforcer) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (e *enforcer) ResponsesC() <-chan *message.ExpiringMessage {
	return nil
}

func generateDeploymentEnforcement(a *storage.Alert) *central.DeploymentEnforcement {
	return &central.DeploymentEnforcement{
		DeploymentId:   a.GetDeployment().GetId(),
		DeploymentName: a.GetDeployment().GetName(),
		DeploymentType: a.GetDeployment().GetType(),
		Namespace:      a.GetDeployment().GetNamespace(),
		AlertId:        a.GetId(),
		PolicyName:     a.GetPolicy().GetName(),
	}
}

func (e *enforcer) ProcessAlertResults(action central.ResourceAction, stage storage.LifecycleStage, alertResults *central.AlertResults) {
	if action != central.ResourceAction_CREATE_RESOURCE {
		return
	}
	for _, a := range alertResults.GetAlerts() {
		if a.GetEnforcement().GetAction() == storage.EnforcementAction_UNSET_ENFORCEMENT {
			continue
		}
		// Do not enforce if there is a bypass annotation specified
		if !enforcers.ShouldEnforce(a.GetDeployment().GetAnnotations()) {
			continue
		}
		switch stage {
		case storage.LifecycleStage_DEPLOY:
			e.actionsC <- &central.SensorEnforcement{
				Enforcement: a.GetEnforcement().Action,
				Resource: &central.SensorEnforcement_Deployment{
					Deployment: generateDeploymentEnforcement(a),
				},
			}
		case storage.LifecycleStage_RUNTIME:
			if numProcesses := len(a.GetProcessViolation().GetProcesses()); numProcesses != 1 {
				log.Errorf("Runtime alert on policy %q and deployment %q has %d process violations. Expected only 1", a.GetPolicy().GetName(), a.GetDeployment().GetName(), numProcesses)
				continue
			}
			e.actionsC <- &central.SensorEnforcement{
				Enforcement: a.GetEnforcement().Action,
				Resource: &central.SensorEnforcement_ContainerInstance{
					ContainerInstance: &central.ContainerInstanceEnforcement{
						PodId:                 a.GetProcessViolation().GetProcesses()[0].GetPodId(),
						DeploymentEnforcement: generateDeploymentEnforcement(a),
					},
				},
			}
		}
	}
}

func (e *enforcer) ProcessMessage(msg *central.MsgToSensor) error {
	enforcement := msg.GetEnforcement()
	if enforcement == nil {
		return nil
	}

	if enforcement.GetEnforcement() == storage.EnforcementAction_UNSET_ENFORCEMENT {
		return errors.Errorf("received enforcement with unset action: %s", proto.MarshalTextString(enforcement))
	}

	select {
	case e.actionsC <- enforcement:
		return nil
	case <-e.stopper.Flow().StopRequested():
		return errors.Errorf("unable to send enforcement: %s", proto.MarshalTextString(enforcement))
	}
}

func (e *enforcer) start() {
	defer e.stopper.Flow().ReportStopped()

	for {
		select {
		case action := <-e.actionsC:
			f, ok := e.enforcementMap[action.Enforcement]
			if !ok {
				log.Errorf("unknown enforcement action: %s", action.Enforcement)
				continue
			}

			if err := f(concurrency.AsContext(e.stopper.LowLevel().GetStopRequestSignal()), action); err != nil {
				log.Errorf("error during enforcement. action: %s err: %v", proto.MarshalTextString(action), err)
			} else {
				log.Infof("enforcement successful. action %s", proto.MarshalTextString(action))
			}
		case <-e.stopper.Flow().StopRequested():
			log.Info("Shutting down Enforcer")
			return
		}
	}
}

func (e *enforcer) Start() error {
	go e.start()
	return nil
}

func (e *enforcer) Stop(_ error) {
	e.stopper.Client().Stop()
	_ = e.stopper.Client().Stopped().Wait()
}

func (e *enforcer) Notify(common.SensorComponentEvent) {}
