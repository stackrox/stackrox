package admissioncontroller

import (
	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/pkg/centralsensor"
	"github.com/stackrox/stackrox/pkg/concurrency"
	"github.com/stackrox/stackrox/sensor/common"
)

// AdmCtrlMsgForwarder returns a wrapper that intercepts messages from sensor components and forwards
// them to Central as well as admission control manager.
type AdmCtrlMsgForwarder interface {
	common.SensorComponent
}

// NewAdmCtrlMsgForwarder returns a new intance of AdmCtrlMsgForwarder.
func NewAdmCtrlMsgForwarder(admCtrlMgr SettingsManager, components ...common.SensorComponent) AdmCtrlMsgForwarder {
	return &admCtrlMsgForwarderImpl{
		admCtrlMgr: admCtrlMgr,
		components: components,

		stopSig:  concurrency.NewSignal(),
		centralC: make(chan *central.MsgFromSensor),
	}
}

type admCtrlMsgForwarderImpl struct {
	admCtrlMgr SettingsManager
	components []common.SensorComponent

	centralC chan *central.MsgFromSensor

	stopSig concurrency.Signal
}

func (h *admCtrlMsgForwarderImpl) Start() error {
	for _, component := range h.components {
		if err := component.Start(); err != nil {
			return err
		}
	}

	go h.run()
	return nil
}

func (h *admCtrlMsgForwarderImpl) Stop(err error) {
	for _, component := range h.components {
		component.Stop(err)
	}

	h.stopSig.Signal()
}

func (h *admCtrlMsgForwarderImpl) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (h *admCtrlMsgForwarderImpl) ProcessMessage(msg *central.MsgToSensor) error {
	return nil
}

func (h *admCtrlMsgForwarderImpl) ResponsesC() <-chan *central.MsgFromSensor {
	return h.centralC
}

func (h *admCtrlMsgForwarderImpl) run() {
	for _, component := range h.components {
		if responsesC := component.ResponsesC(); responsesC != nil {
			go h.forwardResponses(responsesC)
		}
	}
}

func (h *admCtrlMsgForwarderImpl) forwardResponses(from <-chan *central.MsgFromSensor) {
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
			case <-h.stopSig.Done():
			}
		case <-h.stopSig.Done():
			return
		}
	}
}
