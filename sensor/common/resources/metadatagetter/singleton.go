package metadatagetter

import (
	pkgNotifiers "github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/resources/namespaces"
)

var (
	once sync.Once

	metadataGetter *memStoreMetadataGetter
)

func initialize() {
	metadataGetter = newMetadataGetter(namespaces.Singleton())
}

// Singleton returns a MetadataGetter utilizing a memory backed namespace store.
func Singleton() pkgNotifiers.MetadataGetter {
	once.Do(initialize)
	return metadataGetter
}
