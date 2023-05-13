package service

import (
	"context"

	"github.com/stackrox/rox/central/continuousintegration/datastore"
	"github.com/stackrox/rox/central/continuousintegration/token"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the interface to the gRPC service for users.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.ContinuousIntegrationServiceServer
}

// New creates a new instances of the continuous integration service.
func New(dataStore datastore.DataStore, retriever token.Exchanger) Service {
	return &serviceImpl{
		dataStore: dataStore,
		exchanger: retriever,
	}
}
