package collectorruntimeconfig

import (
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/concurrency"

	"github.com/stackrox/rox/pkg/sync"
)

var (
	instance *handlerImpl
	once     sync.Once
)

func newHandler() *handlerImpl {
	return &handlerImpl{
		stopSig:                    concurrency.NewSignal(),
		updateSig:                  concurrency.NewSignal(),
		collectorConfig:            nil,
		collectorConfigProtoStream: concurrency.NewValueStream[*sensor.CollectorConfig](nil),
	}
}

// Singleton returns the singleton instance for the collector runtime config handler.
func Singleton() Handler {
	once.Do(func() {
		instance = newHandler()
	})
	return instance
}

// StoreInstance returns the singleton instance for the collector runtime config store.
func StoreInstance() Store {
	once.Do(func() {
		instance = newHandler()
	})
	return instance
}
