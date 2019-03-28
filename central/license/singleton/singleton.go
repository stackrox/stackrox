package singleton

import (
	"github.com/stackrox/rox/central/license/manager"
	"github.com/stackrox/rox/central/license/store"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	instance     manager.LicenseManager
	instanceInit sync.Once
)

// Singleton returns the license manager singleton instance
func Singleton() manager.LicenseManager {
	instanceInit.Do(func() {
		instance = manager.New(store.Singleton(), validatorInstance)
	})
	return instance
}
