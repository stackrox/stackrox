package namespaceproperties

import (
	"github.com/stackrox/rox/central/notifiers"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	resolver *datastoreNamespaceProperties
)

func initialize() {
	resolver = newNamespaceProperties()
}

func Singleton() notifiers.NamespaceProperties {
	once.Do(initialize)
	return resolver
}
