package injector

import (
	"sync"

	datastore "github.com/stackrox/rox/central/productusage/datastore/securedunits"
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
