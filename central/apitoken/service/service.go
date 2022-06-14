package service

import (
	"context"

	"github.com/stackrox/rox/central/apitoken/backend"
	roleDS "github.com/stackrox/rox/central/role/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the interface to the svc that handles API keys.
type Service interface {
	v1.APITokenServiceServer

	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
}

// New returns a ready-to-use instance of Service.
func New(backend backend.Backend, roles roleDS.DataStore) Service {
	return &serviceImpl{backend: backend, roles: roles}
}
