package metadatagetter

import (
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	resolver *datastoreMetadataGetter
)

func initialize() {
	resolver = newMetadataGetter()
}

// Singleton provides the interface for getting annotation values with a datastore backed implementation.
func Singleton() notifiers.MetadataGetter {
	once.Do(initialize)
	return resolver
}
