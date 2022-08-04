package service

import (
	"context"

	"github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/central/sensor/service/connection"
	watchedImageDataStore "github.com/stackrox/rox/central/watchedimage/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/logging"
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
func New(datastore datastore.DataStore, watchedImages watchedImageDataStore.DataStore, riskManager manager.Manager,
	connManager connection.Manager, enricher enricher.ImageEnricher, metadataCache expiringcache.Cache) Service {
	return &serviceImpl{
		datastore:     datastore,
		watchedImages: watchedImages,
		riskManager:   riskManager,
		enricher:      enricher,
		metadataCache: metadataCache,
		connManager:   connManager,

		internalScanSemaphore: semaphore.NewWeighted(int64(env.MaxParallelImageScanInternal.IntegerSetting())),
	}
}
