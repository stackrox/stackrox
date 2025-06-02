package admissioncontroller

import (
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
)

const (
	alertHandlerComponentName = "alert-handler"
)

var (
	errCentralNoReachable = errors.New("central is not reachable")
)

// AlertHandler forwards the alerts sent by admission control webhook to Central.
//
//go:generate mockgen-wrapper
type AlertHandler interface {
	ProcessAlerts(alerts *sensor.AdmissionControlAlerts) error
	common.SensorComponent
}

type alertHandlerImpl struct {
	output       chan *message.ExpiringMessage
	state        atomic.Value
	stopSig      concurrency.Signal
	centralReady concurrency.Signal
}

var _ common.SensorComponent = (*alertHandlerImpl)(nil)

func (h *alertHandlerImpl) Start() error {
	h.state.Store(common.SensorComponentStateSTARTING)
	go h.run()
	h.state.Store(common.SensorComponentStateSTARTED)
	return nil
}

func (h *alertHandlerImpl) Stop(_ error) {
	h.state.Store(common.SensorComponentStateSTOPPING)
	h.stopSig.Signal()
	h.state.Store(common.SensorComponentStateSTOPPED)
}

func (h *alertHandlerImpl) Notify(e common.SensorComponentEvent) {
	log.Info(common.LogSensorComponentEvent(e))
	switch e {
	case common.SensorComponentEventCentralReachable:
		h.centralReady.Signal()
		h.state.Store(common.SensorComponentStateONLINE)
	case common.SensorComponentEventOfflineMode:
		h.centralReady.Reset()
		h.state.Store(common.SensorComponentStateOFFLINE)
	}
}

func (h *alertHandlerImpl) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (h *alertHandlerImpl) ProcessMessage(_ *central.MsgToSensor) error {
	return nil
}

func (h *alertHandlerImpl) ResponsesC() <-chan *message.ExpiringMessage {
	return h.output
}

func (h *alertHandlerImpl) State() common.SensorComponentState {
	return h.state.Load().(common.SensorComponentState)
}

func (h *alertHandlerImpl) run() {
	<-h.stopSig.Done()
}

func (h *alertHandlerImpl) ProcessAlerts(alerts *sensor.AdmissionControlAlerts) error {
	if !h.centralReady.IsDone() {
		return errCentralNoReachable
	}
	go h.processAlerts(alerts)
	return nil
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

func createAlertResultsMsg(alertResult *central.AlertResults) *message.ExpiringMessage {
	return message.New(&central.MsgFromSensor{
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
	})
}
