package service

import (
	"github.com/stackrox/rox/central/complianceoperator/v2/compliancemanager"
	scanSettingsDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
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
		serviceInstance = New(scanSettingsDS.Singleton(), compliancemanager.Singleton())
	})
	return serviceInstance
}
