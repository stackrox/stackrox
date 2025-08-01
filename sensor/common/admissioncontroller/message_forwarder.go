package admissioncontroller

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
)

// AdmCtrlMsgForwarder returns a wrapper that intercepts messages from sensor components and forwards
// them to Central as well as admission control manager.
type AdmCtrlMsgForwarder interface {
	common.SensorComponent
}

// NewAdmCtrlMsgForwarder returns a new instance of AdmCtrlMsgForwarder.
func NewAdmCtrlMsgForwarder(admCtrlMgr SettingsManager, components ...common.SensorComponent) AdmCtrlMsgForwarder {
	return &admCtrlMsgForwarderImpl{
		admCtrlMgr: admCtrlMgr,
		components: components,

		stopper:  concurrency.NewStopper(),
		centralC: make(chan *message.ExpiringMessage),
	}
}

type admCtrlMsgForwarderImpl struct {
	admCtrlMgr SettingsManager
	components []common.SensorComponent

	centralC chan *message.ExpiringMessage

	stopper concurrency.Stopper
}

func (h *admCtrlMsgForwarderImpl) Start() error {
	for _, component := range h.components {
		if err := component.Start(); err != nil {
			return errors.Wrapf(err, "starting admission controller component %T", component)
		}
	}

	go h.run()
	return nil
}

func (h *admCtrlMsgForwarderImpl) Stop() {
	for _, component := range h.components {
		component.Stop()
	}

	h.stopper.Client().Stop()
}

func (h *admCtrlMsgForwarderImpl) Name() string {
	return "admissioncontroller.admCtrlMsgForwarderImpl"
}

func (h *admCtrlMsgForwarderImpl) Notify(event common.SensorComponentEvent) {
	// Propagate event to sub-components
	for _, c := range h.components {
		c.Notify(event)
	}
}

func (h *admCtrlMsgForwarderImpl) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (h *admCtrlMsgForwarderImpl) ProcessMessage(ctx context.Context, msg *central.MsgToSensor) error {
	errorList := errorhelpers.NewErrorList("ProcessMessage in AdmCtrlMsgForwarder")
	for _, component := range h.components {
		if err := component.ProcessMessage(ctx, msg); err != nil {
			errorList.AddError(err)
		}
	}
	// Wrap any collected errors from forwarding messages
	return errors.Wrap(errorList.ToError(), "processing message in admission control forwarder")
}

func (h *admCtrlMsgForwarderImpl) ResponsesC() <-chan *message.ExpiringMessage {
	return h.centralC
}

func (h *admCtrlMsgForwarderImpl) run() {
	for _, component := range h.components {
		if responsesC := component.ResponsesC(); responsesC != nil {
			go h.forwardResponses(responsesC)
		}
	}
}

func (h *admCtrlMsgForwarderImpl) forwardResponses(from <-chan *message.ExpiringMessage) {
	defer h.stopper.Flow().ReportStopped()
	for {
		select {
		case msg, ok := <-from:
			if !ok {
				return
			}

			if event := msg.GetEvent(); event != nil {
				h.admCtrlMgr.UpdateResources(msg.GetEvent())
			}

			select {
			case h.centralC <- msg:
			case <-h.stopper.Flow().StopRequested():
			}
		case <-h.stopper.Flow().StopRequested():
			return
		}
	}
}
