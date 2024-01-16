package deploymentreconciler

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
)

var (
	log = logging.LoggerForModule()
)

type deploymentReconcilerImpl struct {
	lock        sync.Mutex
	deployments map[string]*storage.Deployment
	toCentral   chan *message.ExpiringMessage
	stopper     concurrency.Stopper
}

func (d *deploymentReconcilerImpl) Notify(e common.SensorComponentEvent) {
	switch e {
	case common.SensorComponentEventSyncFinished:
		d.sendDeleteEvents()
	}
}

func (d *deploymentReconcilerImpl) Start() error {
	return nil
}

func (d *deploymentReconcilerImpl) Stop(_ error) {

}

func (d *deploymentReconcilerImpl) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (d *deploymentReconcilerImpl) ProcessMessage(_ *central.MsgToSensor) error {
	return nil
}

func (d *deploymentReconcilerImpl) ResponsesC() <-chan *message.ExpiringMessage {
	return d.toCentral
}

func (d *deploymentReconcilerImpl) sendDeleteEvents() {
	d.lock.Lock()
	defer d.lock.Unlock()

	log.Infof("Sending %d deleting events", len(d.deployments))
	for _, deployment := range d.deployments {
		log.Infof("Sending delete event for deployment %S:%S", deployment.GetId(), deployment.GetName())
		select {
		case <-d.stopper.Flow().StopRequested():
			return
		case d.toCentral <- message.New(&central.MsgFromSensor{
			Msg: &central.MsgFromSensor_Event{
				Event: &central.SensorEvent{
					Id:     deployment.GetId(),
					Action: central.ResourceAction_REMOVE_RESOURCE,
					Resource: &central.SensorEvent_Deployment{
						Deployment: deployment,
					},
				},
			},
		}):
		}
	}
	d.deployments = make(map[string]*storage.Deployment)
}

func (d *deploymentReconcilerImpl) OnDeploymentRemove(deployment *storage.Deployment) func() {
	d.lock.Lock()
	defer d.lock.Unlock()

	log.Infof("Adding deployment %s:%s", deployment.GetId(), deployment.GetName())
	d.deployments[deployment.GetId()] = deployment
	return func() {
		d.lock.Lock()
		defer d.lock.Unlock()

		delete(d.deployments, deployment.GetId())
	}
}
