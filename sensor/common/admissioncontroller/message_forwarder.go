package admissioncontroller

import (
	"sync"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/sensor/common"
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

		stopper: concurrency.NewStopper(),

		connectionStop: concurrency.NewSignal(),
		forwarderWg:    &sync.WaitGroup{},
	}
}

type admCtrlMsgForwarderImpl struct {
	admCtrlMgr SettingsManager
	components []common.SensorComponent

	centralC chan *central.MsgFromSensor

	stopper concurrency.Stopper

	connectionStop concurrency.Signal
	forwarderWg    *sync.WaitGroup
}

func (h *admCtrlMsgForwarderImpl) Start() error {
	for _, component := range h.components {
		if err := component.Start(); err != nil {
			return err
		}
	}

	return nil
}

func (h *admCtrlMsgForwarderImpl) Stop(err error) {
	for _, component := range h.components {
		component.Stop(err)
	}

	h.stopper.Client().Stop()
}

func (h *admCtrlMsgForwarderImpl) Notify(event common.SensorComponentEvent) {
	// Propagate event to sub-components
	for _, c := range h.components {
		c.Notify(event)
	}

	switch event {
	case common.SensorComponentEventOfflineMode:
		// Any messages written in the old channel must be dropped.
		// The reader of this channel (central sender goroutine) will stop when
		// the connection breaks. This means that no more messages should be read
		// from this channel. By closing it and creating a new one, we are virtually
		// guaranteeing that old messages (prior to the restart) are not considered
		// once the connection is back up.
		h.connectionStop.Signal()

	case common.SensorComponentEventCentralReachable:
		h.connectionStop.Reset()
		// Only create the responses channel when the connection is available
		go h.run()
	}
}

func (h *admCtrlMsgForwarderImpl) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (h *admCtrlMsgForwarderImpl) ProcessMessage(msg *central.MsgToSensor) error {
	errorList := errorhelpers.NewErrorList("ProcessMessage in AdmCtrlMsgForwarder")
	for _, component := range h.components {
		if err := component.ProcessMessage(msg); err != nil {
			errorList.AddError(err)
		}
	}
	return errorList.ToError()
}

func (h *admCtrlMsgForwarderImpl) ResponsesC() <-chan *central.MsgFromSensor {
	return h.centralC
}

func (h *admCtrlMsgForwarderImpl) run() {
	// If the connection restarts too fast, we might try to start new forwarder channels
	// before the old ones were flushed. First we need to make sure that any forwarders
	// in the wait group are done.
	h.forwarderWg.Wait()
	h.centralC = make(chan *central.MsgFromSensor)
	for _, component := range h.components {
		if responsesC := component.ResponsesC(); responsesC != nil {
			go h.forwardResponses(responsesC)
		}
	}
}

func (h *admCtrlMsgForwarderImpl) forwardResponses(from <-chan *central.MsgFromSensor) {
	defer h.stopper.Flow().ReportStopped()
	defer close(h.centralC)
	defer h.forwarderWg.Done()
	h.forwarderWg.Add(1)
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
			case <-h.connectionStop.Done():
				return
			}
		case <-h.stopper.Flow().StopRequested():
			return
		case <-h.connectionStop.Done():
			return
		}
	}
}
