package service

import (
	"github.com/stackrox/rox/central/config/datastore"
	"github.com/stackrox/rox/central/metrics/aggregator"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(datastore.Singleton(), aggregator.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
