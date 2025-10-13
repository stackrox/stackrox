package service

import (
	"context"

	"github.com/stackrox/rox/central/alert/datastore"
	baselineDatastore "github.com/stackrox/rox/central/processbaseline/datastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/logging"
	pkgNotifier "github.com/stackrox/rox/pkg/notifier"
)

var (
	log = logging.LoggerForModule()
)

// Service is a thin facade over a domain layer that handles CRUD use cases on Alert objects from API clients.
type Service interface {
	grpc.APIService
	v1.AlertServiceServer

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
}

// New returns a new Service soleInstance using the given DataStore.
func New(datastore datastore.DataStore, baselines baselineDatastore.DataStore, notifier pkgNotifier.Processor, connectionManager connection.Manager) Service {
	return &serviceImpl{
		dataStore:         datastore,
		notifier:          notifier,
		baselines:         baselines,
		connectionManager: connectionManager,
	}
}
