package injector

import (
	datastore "github.com/stackrox/rox/central/administration/usage/datastore/securedunits"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once     sync.Once
	injector Injector
)

// Singleton returns the injector singleton.
func Singleton() Injector {
	once.Do(func() {
		injector = NewInjector(datastore.Singleton())
	})
	return injector
}
