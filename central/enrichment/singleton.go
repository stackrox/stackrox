package enrichment

import (
	"context"
	"time"

	cveDataStore "github.com/stackrox/rox/central/cve/datastore"
	"github.com/stackrox/rox/central/cve/fetcher"
	imageCVEDataStore "github.com/stackrox/rox/central/cve/image/datastore"
	nodeCVEDataStore "github.com/stackrox/rox/central/cve/node/datastore"
	"github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/imageintegration"
	imageIntegrationDS "github.com/stackrox/rox/central/imageintegration/datastore"
	"github.com/stackrox/rox/central/integrationhealth/reporter"
	signatureIntegrationDataStore "github.com/stackrox/rox/central/signatureintegration/datastore"
	"github.com/stackrox/rox/central/vulnerabilityrequest/suppressor"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/features"
	imageEnricher "github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/metrics"
	nodeEnricher "github.com/stackrox/rox/pkg/nodes/enricher"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ie                    imageEnricher.ImageEnricher
	ne                    nodeEnricher.NodeEnricher
	en                    Enricher
	cf                    fetcher.OrchestratorIstioCVEManager
	manager               Manager
	imageIntegrationStore imageIntegrationDS.DataStore
	metadataCacheOnce     sync.Once
	metadataCache         expiringcache.Cache

	imageCacheExpiryDuration = 4 * time.Hour
)

func initialize() {
	var imageCVESuppressor imageEnricher.CVESuppressor
	var nodeCVESuppressor nodeEnricher.CVESuppressor
	if features.PostgresDatastore.Enabled() {
		imageCVESuppressor = imageCVEDataStore.Singleton()
		nodeCVESuppressor = nodeCVEDataStore.Singleton()
	} else {
		imageCVESuppressor = cveDataStore.Singleton()
		nodeCVESuppressor = cveDataStore.Singleton()
	}

	ie = imageEnricher.New(imageCVESuppressor, suppressor.Singleton(), imageintegration.Set(),
		metrics.CentralSubsystem, ImageMetadataCacheSingleton(), datastore.Singleton().GetImage, reporter.Singleton(),
		signatureIntegrationDataStore.Singleton().GetAllSignatureIntegrations)
	ne = nodeEnricher.New(nodeCVESuppressor, metrics.CentralSubsystem)
	en = New(datastore.Singleton(), ie)
	cf = fetcher.SingletonManager()
	initializeManager()
}

func initializeManager() {
	ctx := sac.WithAllAccess(context.Background())
	manager = newManager(imageintegration.Set(), ne, cf)

	imageIntegrationStore = imageIntegrationDS.Singleton()
	integrations, err := imageIntegrationStore.GetImageIntegrations(ctx, &v1.GetImageIntegrationsRequest{})
	if err != nil {
		log.Errorf("unable to use previous integrations: %s", err)
		return
	}
	for _, ii := range integrations {
		if err := manager.Upsert(ii); err != nil {
			log.Errorf("unable to use previous integration %s: %v", ii.GetName(), err)
		}
	}
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
