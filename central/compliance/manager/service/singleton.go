package service

import (
	"github.com/stackrox/stackrox/central/compliance/manager"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	serviceInstance ComplianceManagementService
	serviceInit     sync.Once
)

// Singleton returns the compliance management service singleton instance.
func Singleton() ComplianceManagementService {
	serviceInit.Do(func() {
		serviceInstance = NewService(manager.Singleton())
	})
	return serviceInstance
}
