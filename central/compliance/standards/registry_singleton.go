package standards

import (
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/compliance/standards/metadata"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	registryInstance     *Registry
	registryInstanceInit sync.Once
)

// RegistrySingleton returns the singleton instance of the compliance standards Registry.
func RegistrySingleton() *Registry {
	registryInstanceInit.Do(func() {
		var err error
		registryInstance, err = NewRegistry(framework.RegistrySingleton(), metadata.AllStandards...)
		utils.CrashOnError(err)
	})
	return registryInstance
}
