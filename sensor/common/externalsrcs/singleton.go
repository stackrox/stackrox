package externalsrcs

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/concurrency"
	"github.com/stackrox/stackrox/pkg/net"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	instance *handlerImpl
	once     sync.Once
)

func newHandler() *handlerImpl {
	return &handlerImpl{
		stopSig:                  concurrency.NewSignal(),
		updateSig:                concurrency.NewSignal(),
		entities:                 make(map[net.IPNetwork]*storage.NetworkEntityInfo),
		entitiesByID:             make(map[string]*storage.NetworkEntityInfo),
		ipNetworkListProtoStream: concurrency.NewValueStream(nil),
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
