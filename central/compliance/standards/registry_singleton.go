package standards

import (
	"sync"

	"github.com/stackrox/rox/central/compliance/standards/index"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	registryInstance     *Registry
	registryInstanceInit sync.Once
)

// RegistrySingleton returns the singleton instance of the compliance standards Registry.
func RegistrySingleton() *Registry {
	registryInstanceInit.Do(func() {
		indexer := index.New(globalindex.GetGlobalIndex())
		registryInstance = NewRegistry(indexer)
		utils.Must(registryInstance.RegisterStandards())
	})
	return registryInstance
}
