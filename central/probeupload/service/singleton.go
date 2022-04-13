package service

import (
	"github.com/stackrox/stackrox/central/probeupload/manager"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	instance     Service
	instanceInit sync.Once
)

// Singleton returns the singleton instance for the probe upload service.
func Singleton() Service {
	instanceInit.Do(func() {
		instance = newService(manager.Singleton())
	})
	return instance
}
