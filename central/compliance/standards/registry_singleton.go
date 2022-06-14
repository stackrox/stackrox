package standards

import (
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/compliance/standards/index"
	"github.com/stackrox/rox/central/compliance/standards/metadata"
	"github.com/stackrox/rox/central/globalindex"
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
		memIndex, err := globalindex.MemOnlyIndex()
		utils.CrashOnError(err)
		indexer := index.New(memIndex)
		registryInstance, err = NewRegistry(indexer, framework.RegistrySingleton(), metadata.AllStandards...)
		utils.CrashOnError(err)
	})
	return registryInstance
}
