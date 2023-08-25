package service

import (
	"context"

	"github.com/stackrox/rox/central/cluster/datastore"
	configDatastore "github.com/stackrox/rox/central/config/datastore"
	"github.com/stackrox/rox/central/probesources"
	"github.com/stackrox/rox/central/risk/manager"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the interface to the microservice that serves alert data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.ClustersServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(
	datastore datastore.DataStore,
	riskManager manager.Manager,
	probeSources probesources.ProbeSources,
	sysConfigDatastore configDatastore.DataStore,
) Service {
	return &serviceImpl{
		datastore:          datastore,
		riskManager:        riskManager,
		probeSources:       probeSources,
		sysConfigDatastore: sysConfigDatastore,
	}
}
