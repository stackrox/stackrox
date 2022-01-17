package enrichment

import (
	"time"

	cveDataStore "github.com/stackrox/rox/central/cve/datastore"
	"github.com/stackrox/rox/central/cve/fetcher"
	"github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/imageintegration"
	"github.com/stackrox/rox/central/integrationhealth/reporter"
	"github.com/stackrox/rox/central/vulnerabilityrequest/suppressor"
	"github.com/stackrox/rox/pkg/expiringcache"
	imageEnricher "github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/metrics"
	nodeEnricher "github.com/stackrox/rox/pkg/nodes/enricher"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ie      imageEnricher.ImageEnricher
	ne      nodeEnricher.NodeEnricher
	en      Enricher
	cf      fetcher.OrchestratorIstioCVEManager
	manager Manager

	imageScanCacheOnce sync.Once
	imageScanCache     expiringcache.Cache

	metadataCacheOnce sync.Once
	metadataCache     expiringcache.Cache

	imageCacheExpiryDuration = 4 * time.Hour
)

func initialize() {
	ie = imageEnricher.New(cveDataStore.Singleton(), suppressor.Singleton(), imageintegration.Set(),
		metrics.CentralSubsystem, ImageMetadataCacheSingleton(), ImageScanCacheSingleton(), reporter.Singleton())
	ne = nodeEnricher.New(cveDataStore.Singleton(), metrics.CentralSubsystem)
	en = New(datastore.Singleton(), ie)
	cf = fetcher.SingletonManager()
	manager = newManager(imageintegration.Set(), ne, cf)
}

// Singleton provides the singleton Enricher to use.
func Singleton() Enricher {
	once.Do(initialize)
	return en
}

// ImageEnricherSingleton provides the singleton ImageEnricher to use.
func ImageEnricherSingleton() imageEnricher.ImageEnricher {
	once.Do(initialize)
	return ie
}

// ImageScanCacheSingleton returns the cache for image scans
func ImageScanCacheSingleton() expiringcache.Cache {
	imageScanCacheOnce.Do(func() {
		imageScanCache = expiringcache.NewExpiringCache(imageCacheExpiryDuration)
	})
	return imageScanCache
}

// ImageMetadataCacheSingleton returns the cache for image metadata
func ImageMetadataCacheSingleton() expiringcache.Cache {
	metadataCacheOnce.Do(func() {
		metadataCache = expiringcache.NewExpiringCache(imageCacheExpiryDuration, expiringcache.UpdateExpirationOnGets)
	})
	return metadataCache
}

// NodeEnricherSingleton provides the singleton NodeEnricher to use.
func NodeEnricherSingleton() nodeEnricher.NodeEnricher {
	once.Do(initialize)
	return ne
}

// ManagerSingleton returns the multiplexing manager
func ManagerSingleton() Manager {
	once.Do(initialize)
	return manager
}
