package service

import (
	"context"

	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	podDS "github.com/stackrox/rox/central/pod/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/postgres"
)

// Service provides the interface to the vulnerability management service.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.VulnMgmtServiceServer
}

// New returns a new vulnerability management service instance.
func New(db postgres.DB, deployments deploymentDS.DataStore, images imageDS.DataStore, pods podDS.DataStore) Service {
	return &serviceImpl{
		db:          db,
		deployments: deployments,
		images:      images,
		pods:        pods,
	}
}
