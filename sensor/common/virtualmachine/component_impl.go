package virtualmachine

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
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

var (
	errCapabilityNotSupported = errors.New("Central does not have virtual machine capability")
	errCentralNotReachable    = errors.New("Central is not reachable")
	errInputChanClosed        = errors.New("channel receiving virtual machines is closed")
	errStartMoreThanOnce      = errors.New("unable to start the component more than once")
)

type componentImpl struct {
	centralReady    concurrency.Signal
	lock            *sync.Mutex
	stopper         concurrency.Stopper
	toCentral       <-chan *message.ExpiringMessage
	virtualMachines chan *sensor.VirtualMachine
}

func (c *componentImpl) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (c *componentImpl) Send(ctx context.Context, vm *sensor.VirtualMachine) error {
	if !c.centralReady.IsDone() {
		log.Warnf("Cannot send virtual machine %q to Central because Central is not reachable", vm.GetId())
		metrics.VirtualMachineSent.With(metrics.StatusCentralNotReadyLabels).Inc()
		return errox.ResourceExhausted.CausedBy(errCentralNotReachable)
	}
	if !centralcaps.Has(centralsensor.VirtualMachinesSupported) {
		return errox.NotImplemented.CausedBy(errCapabilityNotSupported)
	}

	select {
	case <-ctx.Done():
		// Return ResourceExhausted to indicate the client to retry on timeouts.
		if err := ctx.Err(); errors.Is(err, context.DeadlineExceeded) {
			metrics.VirtualMachineSent.With(metrics.StatusTimeoutLabels).Inc()
			return errox.ResourceExhausted.CausedBy(ctx.Err())
		}
		metrics.VirtualMachineSent.With(metrics.StatusErrorLabels).Inc()
		return errors.Wrap(ctx.Err(), "context is done")
	case c.virtualMachines <- vm:
		return nil
	}
}

func (c *componentImpl) Name() string {
	return "virtualMachine.componentImpl"
}

func (c *componentImpl) Notify(e common.SensorComponentEvent) {
	log.Info(common.LogSensorComponentEvent(e))
	switch e {
	case common.SensorComponentEventCentralReachable:
		c.centralReady.Signal()
	case common.SensorComponentEventOfflineMode:
		// As clients are expected to retry virtual machine upserts when Sensor is in
		// offline mode, there is no need to do anything here other than reset the signal.
		c.centralReady.Reset()
	}
}

// ProcessMessage is a no-op because Sensor does not receive any virtual machine data
// from Central.
func (c *componentImpl) ProcessMessage(ctx context.Context, msg *central.MsgToSensor) error {
	return nil
}

// ResponsesC returns a channel with messages to Central. It must be called
// after Start() for the channel to be not nil.
func (c *componentImpl) ResponsesC() <-chan *message.ExpiringMessage {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.toCentral == nil {
		log.Panic("Start must be called before ResponsesC")
	}
	return c.toCentral
}

func (c *componentImpl) Start() error {
	log.Debug("Starting virtual machine component")
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.toCentral != nil {
		return errStartMoreThanOnce
	}
	c.toCentral = c.run()
	return nil
}

func (c *componentImpl) Stop() {
	if !c.stopper.Client().Stopped().IsDone() {
		defer utils.IgnoreError(c.stopper.Client().Stopped().Wait)
	}
	c.stopper.Client().Stop()
}

// run handles the virtual machine data and forwards it to Central.
// This is the only goroutine that writes into the toCentral channel, thus it is
// responsible for creating and closing that chan.
func (c *componentImpl) run() (toCentral <-chan *message.ExpiringMessage) {
	ch2Central := make(chan *message.ExpiringMessage)
	go func() {
		defer func() {
			c.stopper.Flow().ReportStopped()
			close(ch2Central)
		}()
		log.Debugf("virtual machine component is running")
		for {
			select {
			case <-c.stopper.Flow().StopRequested():
				return
			case virtualMachine, ok := <-c.virtualMachines:
				if !ok {
					c.stopper.Flow().StopWithError(errInputChanClosed)
					return
				}
				c.handleVirtualMachine(ch2Central, virtualMachine)
			}
		}
	}()
	return ch2Central
}

func (c *componentImpl) handleVirtualMachine(
	toCentral chan *message.ExpiringMessage,
	virtualMachine *sensor.VirtualMachine,
) {
	log.Debugf("Handling virtual machine %q...", virtualMachine.GetId())
	if virtualMachine == nil {
		log.Warn("Received nil virtual machine: not sending to Central")
		return
	}

	c.sendVirtualMachine(toCentral, virtualMachine)
	metrics.VirtualMachineSent.With(metrics.StatusSuccessLabels).Inc()
}

func (c *componentImpl) sendVirtualMachine(
	toCentral chan<- *message.ExpiringMessage,
	virtualMachine *sensor.VirtualMachine,
) {
	if virtualMachine == nil {
		return
	}
	select {
	case <-c.stopper.Flow().StopRequested():
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
