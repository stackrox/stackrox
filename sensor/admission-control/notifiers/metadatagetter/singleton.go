package metadatagetter

import (
	pkgNotifiers "github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/admission-control/resources/namespaces"
)

var (
	once sync.Once

	metadatagetter *memStoreMetadataGetter
)

func initialize() {
	metadatagetter = newMetadataGetter(namespaces.Singleton())
}

// Singleton provides the interface for getting annotation values with an inmemory store backed implementation.
func Singleton() pkgNotifiers.MetadataGetter {
	once.Do(initialize)
	return metadatagetter
}
