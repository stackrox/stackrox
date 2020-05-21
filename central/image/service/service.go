package service

import (
	"context"

	cveDataStore "github.com/stackrox/rox/central/cve/datastore"
	"github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/risk/manager"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/logging"
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
func New(datastore datastore.DataStore, cveDatastore cveDataStore.DataStore, riskManager manager.Manager, enricher enricher.ImageEnricher, metadataCache, scanCache expiringcache.Cache) Service {
	return &serviceImpl{
		datastore:     datastore,
		cveDatastore:  cveDatastore,
		riskManager:   riskManager,
		enricher:      enricher,
		metadataCache: metadataCache,
		scanCache:     scanCache,
	}
}
