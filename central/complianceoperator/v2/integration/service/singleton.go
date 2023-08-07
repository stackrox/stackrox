package service

import (
	"github.com/stackrox/rox/central/cluster/datastore"
	complianceDS "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	serviceInstance     Service
	serviceInstanceInit sync.Once
)

// Singleton returns the singleton instance of the compliance service.
func Singleton() Service {
	if !features.ComplianceEnhancements.Enabled() {
		return nil
	}

	serviceInstanceInit.Do(func() {
		serviceInstance = New(complianceDS.Singleton(), datastore.Singleton())
	})
	return serviceInstance
}
