package service

import (
	"github.com/stackrox/stackrox/central/cluster/datastore"
	"github.com/stackrox/stackrox/central/compliance/aggregation"
	complianceDS "github.com/stackrox/stackrox/central/compliance/datastore"
	"github.com/stackrox/stackrox/central/compliance/standards"
	"github.com/stackrox/stackrox/central/complianceoperator/manager"
	"github.com/stackrox/stackrox/pkg/sync"
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
