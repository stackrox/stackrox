package externaldb

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service is the interface to the gRPC service for configuring backups when external db is enabled
// This means most of the methods is currently unsupported
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.ExternalBackupServiceServer
}

// New returns a new Service instance using the given DataStore.
func New() Service {
	return &serviceImpl{}
}
