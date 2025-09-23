package service

import (
	"context"

	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	imageV2DS "github.com/stackrox/rox/central/imagev2/datastore"
	podDS "github.com/stackrox/rox/central/pod/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the interface to the vulnerability management service.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.VulnMgmtServiceServer
}

// New returns a new vulnerability management service instance.
func New(deployments deploymentDS.DataStore, images imageDS.DataStore, imagesV2 imageV2DS.DataStore, pods podDS.DataStore) Service {
	return &serviceImpl{
		deployments: deployments,
		images:      images,
		imagesV2:    imagesV2,
		pods:        pods,
	}
}
