package broker

import (
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once     sync.Once
	instance *Broker

	stopSig = concurrency.NewErrorSignal()
)

// Singleton returns the singleton instance of the repository scan broker.
func Singleton() *Broker {
	once.Do(func() {
		instance = New(connection.ManagerSingleton(), &stopSig)
	})
	return instance
}

// Stop signals the broker to stop its background goroutines.
func Stop() {
	stopSig.Signal()
}
