package service

import (
	"context"

	"github.com/stackrox/rox/central/alert/datastore"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/logging"
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
func New(datastore datastore.DataStore, notifier notifierProcessor.Processor) Service {
	return &serviceImpl{
		dataStore: datastore,
		notifier:  notifier,
	}
}
