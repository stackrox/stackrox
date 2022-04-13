package service

import (
	"context"

	"github.com/stackrox/stackrox/central/externalbackups/datastore"
	"github.com/stackrox/stackrox/central/externalbackups/manager"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/grpc"
	"github.com/stackrox/stackrox/pkg/integrationhealth"
	"github.com/stackrox/stackrox/pkg/logging"
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
