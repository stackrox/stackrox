package manager

import (
	"sync"

	"github.com/stackrox/rox/central/networkflow/store"
	"github.com/stackrox/rox/central/sensorevent/service/streamer"
)

var (
	once sync.Once

	sm SensorManager
)

// Singleton provides the instance of the SensorManager interface to use for managing sensor
// connections.
func Singleton() SensorManager {
	once.Do(func() {
		sm = New(streamer.ManagerSingleton(), store.Singleton())
	})
	return sm
}
