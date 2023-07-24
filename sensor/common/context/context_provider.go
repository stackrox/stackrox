package contextprovider

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
)

// ContextProvider Sensor component that provides a common context.
// This context will be cancelled and reset if Sensor enters Offline mode.
type ContextProvider interface {
	common.SensorComponent
	GetContext() context.Context
}

var _ ContextProvider = (*contextProviderImpl)(nil)

// NewContextProvider Initializes a new ContextProvider
func NewContextProvider() ContextProvider {
	return &contextProviderImpl{
		centralReachable: concurrency.NewSignal(),
		stopper:          concurrency.NewStopper(),
	}
}

type contextProviderImpl struct {
	cancelContextFn  func()
	sensorContext    context.Context
	centralReachable concurrency.Signal
	stopper          concurrency.Stopper
}

// GetContext returns the sensor context. This call will block until central is reachable.
// This blocking in behavior is needed to ensure unset or old contexts are not passed to other component.
func (c *contextProviderImpl) GetContext() context.Context {
	select {
	case <-c.centralReachable.Done():
		return c.sensorContext
	case <-c.stopper.Flow().StopRequested():
		return nil
	}
}

func (c *contextProviderImpl) Start() error {
	return nil
}

func (c *contextProviderImpl) Stop(_ error) {
	c.stopper.Client().Stop()
}

func (c *contextProviderImpl) Notify(event common.SensorComponentEvent) {
	switch event {
	case common.SensorComponentEventCentralReachable:
		c.sensorContext, c.cancelContextFn = context.WithCancel(context.Background())
		c.centralReachable.Signal()
	case common.SensorComponentEventOfflineMode:
		c.centralReachable.Reset()
		c.cancelContextFn()
	}
}

func (c *contextProviderImpl) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (c *contextProviderImpl) ProcessMessage(_ *central.MsgToSensor) error {
	return nil
}

func (c *contextProviderImpl) ResponsesC() <-chan *message.ExpiringMessage {
	return nil
}
