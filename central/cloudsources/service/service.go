package service

import (
	"context"

	"github.com/stackrox/rox/central/cloudsources/datastore"
	"github.com/stackrox/rox/central/cloudsources/manager"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/cloudsources"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the interface to the gRPC service for users.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.CloudSourcesServiceServer
}

func newService(datastore datastore.DataStore, manager manager.Manager) Service {
	return &serviceImpl{
		ds:            datastore,
		mgr:           manager,
		clientFactory: cloudsources.NewClientForCloudSource,
	}
}
