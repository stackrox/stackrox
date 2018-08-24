package service

import (
	"sync"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/listeners"
)

var (
	once sync.Once

	as Service
)

// newService creates a new streaming service with the collector. It should only be called once.
func newService() Service {
	return &serviceImpl{
		queue:      make(chan *v1.Signal, maxBufferSize),
		indicators: make(chan *listeners.EventWrap),
	}
}

func initialize() {
	as = newService()
}

// Singleton implements a singleton for the client streaming gRPC service between collector and sensor
func Singleton() Service {
	once.Do(initialize)
	return as
}
