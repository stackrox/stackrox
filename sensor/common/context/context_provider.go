package contextprovider

import (
	"context"
	"sync/atomic"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/sync"
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

// NewContextProvider initializes a new ContextProvider
func NewContextProvider() ContextProvider {
	ret := &contextProviderImpl{
		firstCentralConnection: &atomic.Bool{},
	}
	// We initialize here and not in Start to avoid races with other sensor components
	// that might call GetContext on their respective Start functions.
	ret.init(func() (context.Context, func()) {
		return context.WithCancel(context.Background())
	})
	return ret
}

type contextProviderImpl struct {
	mu              sync.RWMutex
	cancelContextFn func()
	sensorContext   context.Context
	// newContextFn defines the function that creates a new context/cancelFunction.
	// This is needed for testing purposes.
	newContextFn func() (context.Context, func())
	// firstCentralConnection indicates whether it is the first time Central is reachable or not.
	// This is needed to make sure any SensorComponent that calls GetContext on Start gets a
	// valid context that won't be overriden on receiving the CentralReachable notification.
	firstCentralConnection *atomic.Bool
}

// GetContext returns the sensor context.
func (c *contextProviderImpl) GetContext() context.Context {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sensorContext
}

// init sets the newContextFn and initializes the sensorContext and the cancelContextFn
func (c *contextProviderImpl) init(ctxFn func() (context.Context, func())) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.newContextFn = ctxFn
	c.sensorContext, c.cancelContextFn = c.newContextFn()
}

func (c *contextProviderImpl) Start() error {
	return nil
}

func (c *contextProviderImpl) Stop(_ error) {}

func (c *contextProviderImpl) Notify(event common.SensorComponentEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch event {
	case common.SensorComponentEventCentralReachable:
		// We only re-create the context if it's not the first time central is reachable.
		if !c.firstCentralConnection.CompareAndSwap(false, true) {
			c.sensorContext, c.cancelContextFn = c.newContextFn()
		}
	case common.SensorComponentEventOfflineMode:
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
