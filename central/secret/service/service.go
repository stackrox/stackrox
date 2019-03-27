package service

import (
	"context"

	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/secret/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the interface to the microservice that serves secret data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.SecretServiceServer
}

// New returns a new Service instance using the given DB and index.
func New(secrets datastore.DataStore, deployments deploymentDataStore.DataStore) Service {
	return &serviceImpl{
		storage:     secrets,
		deployments: deployments,
	}
}
