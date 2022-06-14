package service

import (
	"context"

	"github.com/stackrox/stackrox/central/alert/datastore"
	notifierProcessor "github.com/stackrox/stackrox/central/notifier/processor"
	baselineDatastore "github.com/stackrox/stackrox/central/processbaseline/datastore"
	"github.com/stackrox/stackrox/central/sensor/service/connection"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/grpc"
	"github.com/stackrox/stackrox/pkg/logging"
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
func New(datastore datastore.DataStore, baselines baselineDatastore.DataStore, notifier notifierProcessor.Processor, connectionManager connection.Manager) Service {
	return &serviceImpl{
		dataStore:         datastore,
		notifier:          notifier,
		baselines:         baselines,
		connectionManager: connectionManager,
	}
}
