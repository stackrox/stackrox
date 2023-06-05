package service

import (
	"github.com/stackrox/rox/central/declarativeconfig/health/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ds Service
)

func initialize() {
	ds = New(datastore.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return ds
}
