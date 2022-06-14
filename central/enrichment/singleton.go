package enrichment

import (
	"time"

	cveDataStore "github.com/stackrox/rox/central/cve/datastore"
	"github.com/stackrox/rox/central/cve/fetcher"
	imageCVEDataStore "github.com/stackrox/rox/central/cve/image/datastore"
	"github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/imageintegration"
	"github.com/stackrox/rox/central/integrationhealth/reporter"
	signatureIntegrationDataStore "github.com/stackrox/rox/central/signatureintegration/datastore"
	"github.com/stackrox/rox/central/vulnerabilityrequest/suppressor"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/features"
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

	metadataCacheOnce sync.Once
	metadataCache     expiringcache.Cache

	imageCacheExpiryDuration = 4 * time.Hour
)

func initialize() {
	var imageCVEDS imageCVEDataStore.DataStore
	if features.PostgresDatastore.Enabled() {
		imageCVEDS = imageCVEDataStore.Singleton()
	} else {
		imageCVEDS = cveDataStore.Singleton()
	}

	ie = imageEnricher.New(imageCVEDS, suppressor.Singleton(), imageintegration.Set(),
		metrics.CentralSubsystem, ImageMetadataCacheSingleton(), datastore.Singleton().GetImage, reporter.Singleton(),
		signatureIntegrationDataStore.Singleton().GetAllSignatureIntegrations)
	// TODO: Attach node cve datastore.
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
