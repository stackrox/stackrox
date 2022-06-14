package service

import (
	"github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/compliance/aggregation"
	complianceDS "github.com/stackrox/rox/central/compliance/datastore"
	"github.com/stackrox/rox/central/compliance/standards"
	"github.com/stackrox/rox/central/complianceoperator/manager"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	serviceInstance     Service
	serviceInstanceInit sync.Once
)

// Singleton returns the singleton instance of the compliance service.
func Singleton() Service {
	serviceInstanceInit.Do(func() {
		serviceInstance = New(aggregation.Singleton(), complianceDS.Singleton(), standards.RegistrySingleton(), datastore.Singleton(), manager.Singleton())
	})
	return serviceInstance
}
