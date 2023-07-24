package contextprovider

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
)

type ContextProvider interface {
	common.SensorComponent
	GetContext() context.Context
}

var _ ContextProvider = (*contextProviderImpl)(nil)

func NewContextProvider() ContextProvider {
	return &contextProviderImpl{}
}

type contextProviderImpl struct {
	centralReachable concurrency.Signal
	sensorContext    context.Context
	cancelContextFn  func()
}

// GetContext returns the sensor context. This call will block until central is reachable.
// This blocking in behavior is needed to ensure unset or old contexts are not passed to other component.
func (c *contextProviderImpl) GetContext() context.Context {
	return nil
}

func (c *contextProviderImpl) Start() error {
	return nil
}

func (c *contextProviderImpl) Stop(_ error) {}

func (c *contextProviderImpl) Notify(_ common.SensorComponentEvent) {}

func (c *contextProviderImpl) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (c *contextProviderImpl) ProcessMessage(_ *central.MsgToSensor) error {
	return nil
}

func (c *contextProviderImpl) ResponsesC() <-chan *message.ExpiringMessage {
	return nil
}
