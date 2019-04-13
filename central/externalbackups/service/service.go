package service

import (
	"context"

	"github.com/stackrox/rox/central/externalbackups/manager"
	backupStore "github.com/stackrox/rox/central/externalbackups/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Service is the interface to the gRPC service for configuring backups
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.ExternalBackupServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(store backupStore.Store, manager manager.Manager) Service {
	return &serviceImpl{
		store:   store,
		manager: manager,
	}
}
