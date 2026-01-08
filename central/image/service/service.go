package service

import (
	"context"

	"github.com/stackrox/rox/central/administration/events"
	"github.com/stackrox/rox/central/image/datastore"
	imageV2Datastore "github.com/stackrox/rox/central/imagev2/datastore"
	"github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/central/role/sachelper"
	"github.com/stackrox/rox/central/sensor/service/connection"
	watchedImageDataStore "github.com/stackrox/rox/central/watchedimage/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/images"
	"github.com/stackrox/rox/pkg/images/cache"
	"github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/waiter"
	"golang.org/x/sync/semaphore"
)

var (
	log = logging.LoggerForModule(events.EnableAdministrationEvents())
)

// Service provides the interface to the microservice that serves alert data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.ImageServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(
	datastore datastore.DataStore,
	datastoreV2 imageV2Datastore.DataStore,
	mappingDatastore datastore.DataStore,
	watchedImages watchedImageDataStore.DataStore,
	riskManager manager.Manager,
	connManager connection.Manager,
	enricher enricher.ImageEnricher,
	enricherV2 enricher.ImageEnricherV2,
	metadataCache cache.ImageMetadata,
	scanWaiterManager waiter.Manager[*storage.Image],
	scanWaiterManagerV2 waiter.Manager[*storage.ImageV2],
	clusterSACHelper sachelper.ClusterSacHelper,
) Service {
	images.SetCentralScanSemaphoreLimit(float64(env.MaxParallelImageScanInternal.IntegerSetting()))
	return &serviceImpl{
		datastore:             datastore,
		datastoreV2:           datastoreV2,
		mappingDatastore:      mappingDatastore,
		watchedImages:         watchedImages,
		riskManager:           riskManager,
		enricher:              enricher,
		enricherV2:            enricherV2,
		metadataCache:         metadataCache,
		connManager:           connManager,
		scanWaiterManager:     scanWaiterManager,
		scanWaiterManagerV2:   scanWaiterManagerV2,
		internalScanSemaphore: semaphore.NewWeighted(int64(env.MaxParallelImageScanInternal.IntegerSetting())),
		clusterSACHelper:      clusterSACHelper,
	}
}
