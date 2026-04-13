package externalsrcs

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
		stopSig:                  concurrency.NewSignal(),
		updateSig:                concurrency.NewSignal(),
		ipNetworkListProtoStream: concurrency.NewValueStream[*sensor.IPNetworkList](nil),
		// entities and entitiesByID are nil — built lazily on first lookup.
	}
}

// Singleton returns the singleton instance for the network graph external sources handler.
func Singleton() Handler {
	once.Do(func() {
		instance = newHandler()
	})
	return instance
}

// StoreInstance returns the singleton instance for the network graph external sources store.
func StoreInstance() Store {
	once.Do(func() {
		instance = newHandler()
	})
	return instance
}
