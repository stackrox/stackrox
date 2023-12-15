package service

import (
	"context"

	"github.com/stackrox/rox/central/administration/events"
	"github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/central/role/sachelper"
	"github.com/stackrox/rox/central/sensor/service/connection"
	watchedImageDataStore "github.com/stackrox/rox/central/watchedimage/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/images/integration"
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
	watchedImages watchedImageDataStore.DataStore,
	riskManager manager.Manager,
	connManager connection.Manager,
	enricher enricher.ImageEnricher,
	metadataCache expiringcache.Cache,
	scanWaiterManager waiter.Manager[*storage.Image],
	clusterSACHelper sachelper.ClusterSacHelper,
	integrationsSet integration.Set,
) Service {

	return &serviceImpl{
		datastore:             datastore,
		watchedImages:         watchedImages,
		riskManager:           riskManager,
		enricher:              enricher,
		metadataCache:         metadataCache,
		connManager:           connManager,
		scanWaiterManager:     scanWaiterManager,
		internalScanSemaphore: semaphore.NewWeighted(int64(env.MaxParallelImageScanInternal.IntegerSetting())),
		clusterSACHelper:      clusterSACHelper,
		integrationSet:        integrationsSet,
	}
}
