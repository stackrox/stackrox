package admissioncontroller

import (
	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/generated/internalapi/sensor"
	"github.com/stackrox/stackrox/pkg/centralsensor"
	"github.com/stackrox/stackrox/pkg/concurrency"
	"github.com/stackrox/stackrox/sensor/common"
)

// AlertHandler forwards the alerts sent by admission control webhook to Central.
type AlertHandler interface {
	ProcessAlerts(alerts *sensor.AdmissionControlAlerts)
	common.SensorComponent
}

type alertHandlerImpl struct {
	output  chan *central.MsgFromSensor
	stopSig concurrency.Signal
}

func (h *alertHandlerImpl) Start() error {
	go h.run()
	return nil
}

func (h *alertHandlerImpl) Stop(_ error) {
	h.stopSig.Signal()
}

func (h *alertHandlerImpl) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (h *alertHandlerImpl) ProcessMessage(msg *central.MsgToSensor) error {
	return nil
}

func (h *alertHandlerImpl) ResponsesC() <-chan *central.MsgFromSensor {
	return h.output
}

func (h *alertHandlerImpl) run() {
	<-h.stopSig.Done()
}

func (h *alertHandlerImpl) ProcessAlerts(alerts *sensor.AdmissionControlAlerts) {
	go h.processAlerts(alerts)
}

func (h *alertHandlerImpl) processAlerts(alertMsg *sensor.AdmissionControlAlerts) {
	// Enforcement is carried out by admission controller, hence skip processing it.
	for _, alertResult := range alertMsg.GetAlertResults() {
		select {
		case <-h.stopSig.Done():
			return
		case h.output <- createAlertResultsMsg(alertResult):
		}
	}
}

func createAlertResultsMsg(alertResult *central.AlertResults) *central.MsgFromSensor {
	return &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id: alertResult.GetDeploymentId(),
				Resource: &central.SensorEvent_AlertResults{
					AlertResults: &central.AlertResults{
						DeploymentId: alertResult.GetDeploymentId(),
						Alerts:       alertResult.GetAlerts(),
						Stage:        alertResult.GetStage(),
					},
				},
			},
		},
	}
}
