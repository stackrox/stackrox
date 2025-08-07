package virtualmachine

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/virtualmachine/metrics"
)

// TODO: Buffer has been decreased for testing. Increase the buffer again.
const virtualMachineBufferedChannelSize = 1

var (
	errCapabilityNotSupported = errors.New("Central does not have virtual machine capability")
	errCentralNotReachable    = errors.New("Central is not reachable")
	errInputChanClosed        = errors.New("channel receiving virtual machines is closed")
	errStartMoreThanOnce      = errors.New("unable to start the handler more than once")
)

type handlerImpl struct {
	centralReady concurrency.Signal
	// lock prevents the race condition between Start() [writer] and ResponsesC(), Send() [reader].
	lock            *sync.RWMutex
	stopper         concurrency.Stopper
	toCentral       <-chan *message.ExpiringMessage
	virtualMachines chan *central.VirtualMachine
}

func (h *handlerImpl) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (h *handlerImpl) Send(ctx context.Context, vm *central.VirtualMachine) error {
	if !h.centralReady.IsDone() {
		log.Warnf("Cannot send virtual machine %q to Central because Central is not reachable", vm.GetId())
		metrics.VirtualMachineSent.With(metrics.StatusCentralNotReadyLabels).Inc()
		return errox.ResourceExhausted.CausedBy(errCentralNotReachable)
	}
	if !centralcaps.Has(centralsensor.VirtualMachinesSupported) {
		return errox.NotImplemented.CausedBy(errCapabilityNotSupported)
	}

	h.lock.RLock()
	defer h.lock.RUnlock()
	select {
	case <-ctx.Done():
		if err := ctx.Err(); errors.Is(err, context.DeadlineExceeded) {
			metrics.VirtualMachineSent.With(metrics.StatusTimeoutLabels).Inc()
			return err //nolint:wrapcheck
		}
		metrics.VirtualMachineSent.With(metrics.StatusErrorLabels).Inc()
		return ctx.Err() //nolint:wrapcheck
	case h.virtualMachines <- vm:
		return nil
	}
}

func (h *handlerImpl) Name() string {
	return "virtualMachine.handlerImpl"
}

func (h *handlerImpl) Notify(e common.SensorComponentEvent) {
	log.Info(common.LogSensorComponentEvent(e))
	switch e {
	case common.SensorComponentEventCentralReachable:
		h.centralReady.Signal()
	case common.SensorComponentEventOfflineMode:
		// As clients are expected to retry virtual machine upserts when Sensor is in
		// offline mode, there is no need to do anything here other than reset the signal.
		h.centralReady.Reset()
	}
}

// ProcessMessage is a no-op because Sensor does not receive any virtual machine data
// from Central.
func (h *handlerImpl) ProcessMessage(ctx context.Context, msg *central.MsgToSensor) error {
	return nil
}

// ResponsesC returns a channel with messages to Central. It must be called
// after Start() for the channel to be not nil.
func (h *handlerImpl) ResponsesC() <-chan *message.ExpiringMessage {
	h.lock.RLock()
	defer h.lock.RUnlock()
	if h.toCentral == nil {
		log.Panic("Start must be called before ResponsesC")
	}
	return h.toCentral
}

func (h *handlerImpl) Start() error {
	log.Debug("Starting virtual machine handler")
	h.lock.Lock()
	defer h.lock.Unlock()
	if h.toCentral != nil || h.virtualMachines != nil {
		return errStartMoreThanOnce
	}
	h.virtualMachines = make(chan *central.VirtualMachine, virtualMachineBufferedChannelSize)
	h.toCentral = h.run()
	return nil
}

func (h *handlerImpl) Stop() {
	close(h.virtualMachines)
	if !h.stopper.Client().Stopped().IsDone() {
		defer utils.IgnoreError(h.stopper.Client().Stopped().Wait)
	}
	h.stopper.Client().Stop()
}

// run handles the virtual machine data and forwards it to Central.
// This is the only goroutine that writes into the toCentral channel, thus it is
// responsible for creating and closing that chan.
func (h *handlerImpl) run() (toCentral <-chan *message.ExpiringMessage) {
	ch2Central := make(chan *message.ExpiringMessage)
	go func() {
		defer func() {
			h.stopper.Flow().ReportStopped()
			close(ch2Central)
		}()
		log.Debugf("virtual machine handler is running")
		for {
			select {
			case <-h.stopper.Flow().StopRequested():
				return
			case virtualMachine, ok := <-h.virtualMachines:
				if !ok {
					h.stopper.Flow().StopWithError(errInputChanClosed)
					return
				}
				h.handleVirtualMachine(ch2Central, virtualMachine)
			}
		}
	}()
	return ch2Central
}

func (h *handlerImpl) handleVirtualMachine(
	toCentral chan *message.ExpiringMessage,
	virtualMachine *central.VirtualMachine,
) {
	log.Debugf("Handling virtual machine %q...", virtualMachine.GetId())
	if virtualMachine == nil {
		log.Warn("Received nil virtual machine: not sending to Central")
		return
	}

	h.sendVirtualMachine(toCentral, virtualMachine)
	metrics.VirtualMachineSent.With(metrics.StatusSuccessLabels).Inc()
}

func (h *handlerImpl) sendVirtualMachine(
	toCentral chan<- *message.ExpiringMessage,
	virtualMachine *central.VirtualMachine,
) {
	if virtualMachine == nil {
		return
	}
	select {
	case <-h.stopper.Flow().StopRequested():
	case toCentral <- message.New(&central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     virtualMachine.GetId(),
				Action: central.ResourceAction_SYNC_RESOURCE,
				Resource: &central.SensorEvent_VirtualMachine{
					VirtualMachine: virtualMachine,
				},
			},
		},
	}):
	}
}
