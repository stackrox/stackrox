package service

import (
	"context"

	notifierDataStore "github.com/stackrox/stackrox/central/notifier/datastore"
	"github.com/stackrox/stackrox/central/reportconfigurations/datastore"
	"github.com/stackrox/stackrox/central/reports/manager"
	accessScopeStore "github.com/stackrox/stackrox/central/role/datastore"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/grpc"
)

// Service provides the interface to the gRPC service for roles.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
	v1.ReportConfigurationServiceServer
}

// New returns a new instance of the service. Please use the Singleton instead.
func New(reportConfigStore datastore.DataStore, notifierStore notifierDataStore.DataStore, accessScopeStore accessScopeStore.DataStore, manager manager.Manager) Service {
	return &serviceImpl{
		manager:           manager,
		reportConfigStore: reportConfigStore,
		notifierStore:     notifierStore,
		accessScopeStore:  accessScopeStore,
	}
}
