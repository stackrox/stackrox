package service

import (
	"github.com/stackrox/rox/pkg/sync"

	"github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/compliance/aggregation"
	"github.com/stackrox/rox/central/compliance/standards"
	"github.com/stackrox/rox/central/compliance/store"
)

var (
	serviceInstance     Service
	serviceInstanceInit sync.Once
)

// Singleton returns the singleton instance of the compliance service.
func Singleton() Service {
	serviceInstanceInit.Do(func() {
		serviceInstance = New(aggregation.Singleton(), store.Singleton(), standards.RegistrySingleton(), datastore.Singleton())
	})
	return serviceInstance
}
