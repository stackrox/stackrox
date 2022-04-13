package service

import (
	"context"

	deploymentDS "github.com/stackrox/stackrox/central/deployment/datastore"
	secretDS "github.com/stackrox/stackrox/central/secret/datastore"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/grpc"
)

// Service provides the interface to the microservice that serves secret data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.SecretServiceServer
}

// New returns a new Service instance using the given DB and index.
func New(secrets secretDS.DataStore, deployments deploymentDS.DataStore) Service {
	return &serviceImpl{
		secrets:     secrets,
		deployments: deployments,
	}
}
