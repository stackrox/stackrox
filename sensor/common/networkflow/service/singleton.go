package service

import (
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/networkflow/manager"
)

var (
	once sync.Once

	as Service
)

// newService creates a new streaming service with the collector. It should only be called once.
func newService() Service {
	return &serviceImpl{
		manager: manager.Singleton(),
	}

}

func initialize() {
	// Creates the signal service with the pending cache embedded
	as = newService()
}

// Singleton implements a singleton for the client streaming gRPC service between collector and sensor
func Singleton() Service {
	once.Do(initialize)
	return as
}
