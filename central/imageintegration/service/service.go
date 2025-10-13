package service

import (
	"context"

	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/enrichment"
	"github.com/stackrox/rox/central/imageintegration/datastore"
	"github.com/stackrox/rox/central/reprocessor"
	"github.com/stackrox/rox/central/sensor/service/connection"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/nodes/enricher"
	"github.com/stackrox/rox/pkg/registries"
	"github.com/stackrox/rox/pkg/scanners"
)

var (
	log = logging.LoggerForModule()
)

// Service provides the interface to the microservice that serves alert data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.ImageIntegrationServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(registryFactory registries.Factory,
	scannerFactory scanners.Factory,
	integrationManager enrichment.Manager,
	nodeEnricher enricher.NodeEnricher,
	datastore datastore.DataStore,
	clusterDatastore clusterDatastore.DataStore,
	reprocessorLoop reprocessor.Loop,
	connManager connection.Manager) Service {
	return &serviceImpl{
		registryFactory:    registryFactory,
		scannerFactory:     scannerFactory,
		nodeEnricher:       nodeEnricher,
		integrationManager: integrationManager,
		datastore:          datastore,
		clusterDatastore:   clusterDatastore,
		reprocessorLoop:    reprocessorLoop,
		connManager:        connManager,
	}
}
