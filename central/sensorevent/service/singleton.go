package service

import (
	"sync"

	"github.com/stackrox/rox/central/sensorevent/service/pipeline/all"
	sensorEventDataStore "github.com/stackrox/rox/central/sensorevent/store"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(sensorEventDataStore.Singleton(), all.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
