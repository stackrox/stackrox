package standards

import "sync"

var (
	registryInstance     *Registry
	registryInstanceInit sync.Once
)

// RegistrySingleton returns the singleton instance of the compliance standards Registry.
func RegistrySingleton() *Registry {
	registryInstanceInit.Do(func() {
		registryInstance = NewRegistry()
	})
	return registryInstance
}
