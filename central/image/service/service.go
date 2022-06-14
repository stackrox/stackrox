package service

import (
	"context"

	cveDataStore "github.com/stackrox/stackrox/central/cve/image/datastore"
	"github.com/stackrox/stackrox/central/image/datastore"
	"github.com/stackrox/stackrox/central/risk/manager"
	"github.com/stackrox/stackrox/central/sensor/service/connection"
	watchedImageDataStore "github.com/stackrox/stackrox/central/watchedimage/datastore"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/env"
	"github.com/stackrox/stackrox/pkg/expiringcache"
	"github.com/stackrox/stackrox/pkg/grpc"
	"github.com/stackrox/stackrox/pkg/images/enricher"
	"github.com/stackrox/stackrox/pkg/logging"
	"golang.org/x/sync/semaphore"
)

var (
	log = logging.LoggerForModule()
)

// Service provides the interface to the microservice that serves alert data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.ImageServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(datastore datastore.DataStore, cveDatastore cveDataStore.DataStore, watchedImages watchedImageDataStore.DataStore, riskManager manager.Manager,
	connManager connection.Manager, enricher enricher.ImageEnricher, metadataCache expiringcache.Cache) Service {
	return &serviceImpl{
		datastore:     datastore,
		cveDatastore:  cveDatastore,
		watchedImages: watchedImages,
		riskManager:   riskManager,
		enricher:      enricher,
		metadataCache: metadataCache,
		connManager:   connManager,

		internalScanSemaphore: semaphore.NewWeighted(int64(env.MaxParallelImageScanInternal.IntegerSetting())),
	}
}
