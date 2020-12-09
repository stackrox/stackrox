package service

import (
	"context"

	"github.com/stackrox/rox/central/externalbackups/datastore"
	"github.com/stackrox/rox/central/externalbackups/manager"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/integrationhealth"
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
func New(dataStore datastore.DataStore, reporter integrationhealth.Reporter, manager manager.Manager) Service {
	return &serviceImpl{
		dataStore: dataStore,
		reporter:  reporter,
		manager:   manager,
	}
}
