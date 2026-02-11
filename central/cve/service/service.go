package service

import (
	"context"

	clusterCVEDatastore "github.com/stackrox/rox/central/cve/cluster/datastore"
	imageCVEDatastore "github.com/stackrox/rox/central/cve/image/v2/datastore"
	nodeCVEDatastore "github.com/stackrox/rox/central/cve/node/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the interface to the microservice for CVE metadata.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.CVEServiceServer
}

// New returns a new Service instance using the given DataStores.
func New(imageDataStore imageCVEDatastore.DataStore, nodeDataStore nodeCVEDatastore.DataStore, clusterDataStore clusterCVEDatastore.DataStore) Service {
	return &serviceImpl{
		imageCVEs:   imageDataStore,
		nodeCVEs:    nodeDataStore,
		clusterCVEs: clusterDataStore,
	}
}
