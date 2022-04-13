package service

import (
	"context"

	clusterDatastore "github.com/stackrox/stackrox/central/cluster/datastore"
	"github.com/stackrox/stackrox/central/enrichment"
	"github.com/stackrox/stackrox/central/imageintegration/datastore"
	"github.com/stackrox/stackrox/central/reprocessor"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/grpc"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/nodes/enricher"
	"github.com/stackrox/stackrox/pkg/registries"
	"github.com/stackrox/stackrox/pkg/scanners"
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
	reprocessorLoop reprocessor.Loop) Service {
	return &serviceImpl{
		registryFactory:    registryFactory,
		scannerFactory:     scannerFactory,
		nodeEnricher:       nodeEnricher,
		integrationManager: integrationManager,
		datastore:          datastore,
		clusterDatastore:   clusterDatastore,
		reprocessorLoop:    reprocessorLoop,
	}
}
