package sensornetworkflow

import (
	"sync"

	"github.com/stackrox/rox/central/networkflow/store"
)

var (
	once    sync.Once
	service Service
)

func initialize() {
	service = New(store.Singleton())
}

// Singleton provides the interface for non-service external interaction.
func Singleton() Service {
	once.Do(initialize)
	return service
}
