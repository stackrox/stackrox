package service

import (
	"context"

	"github.com/stackrox/rox/central/lightspeed/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the Lightspeed gRPC API.
type Service interface {
	grpc.APIService
	v1.LightspeedServiceServer

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
}

// New creates a new Lightspeed service.
func New(ds datastore.DataStore) Service {
	return &serviceImpl{
		datastore: ds,
	}
}
