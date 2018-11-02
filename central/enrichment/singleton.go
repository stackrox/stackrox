package enrichment

import (
	"sync"
	"time"

	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/imageintegration"
	imageintegrationDataStore "github.com/stackrox/rox/central/imageintegration/datastore"
	multiplierStore "github.com/stackrox/rox/central/multiplier/store"
	"github.com/stackrox/rox/central/risk"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	once sync.Once

	ie enricher.ImageEnricher
	en Enricher

	scanCacheOnce sync.Once
	scanCache     expiringcache.Cache

	metadataCacheOnce sync.Once
	metadataCache     expiringcache.Cache

	imageCachesDataSize      = 50000
	imageCacheExpiryDuration = 1 * time.Hour
)

func initialize() {
	ie = enricher.New(imageintegration.Set(), metrics.CentralSubsystem, ImageMetadataCacheSingleton(), ImageScanCacheSingleton())

	var err error
	if en, err = New(deploymentDataStore.Singleton(),
		imageDataStore.Singleton(),
		imageintegrationDataStore.Singleton(),
		multiplierStore.Singleton(),
		ie,
		risk.GetScorer()); err != nil {
		panic(err)
	}
}

// Singleton provides the singleton Enricher to use.
func Singleton() Enricher {
	once.Do(initialize)
	return en
}

// ImageEnricherSingleton provides the singleton ImageEnricher to use.
func ImageEnricherSingleton() enricher.ImageEnricher {
	once.Do(initialize)
	return ie
}

// ImageScanCacheSingleton returns the cache for image scans
func ImageScanCacheSingleton() expiringcache.Cache {
	scanCacheOnce.Do(func() {
		scanCache = expiringcache.NewExpiringCacheOrPanic(imageCachesDataSize, imageCacheExpiryDuration)
	})
	return scanCache
}

// ImageMetadataCacheSingleton returns the cache for image metadata
func ImageMetadataCacheSingleton() expiringcache.Cache {
	metadataCacheOnce.Do(func() {
		metadataCache = expiringcache.NewExpiringCacheOrPanic(imageCachesDataSize, imageCacheExpiryDuration)
	})
	return metadataCache
}
