package service

import (
	"context"

	"github.com/stackrox/rox/central/detection"
	"github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/notifier/processor"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Service provides the interface to the microservice that serves alert data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	GetNotifier(ctx context.Context, request *v1.ResourceByID) (*storage.Notifier, error)
	GetNotifiers(ctx context.Context, request *v1.GetNotifiersRequest) (*v1.GetNotifiersResponse, error)
	PutNotifier(ctx context.Context, request *storage.Notifier) (*v1.Empty, error)
	PostNotifier(ctx context.Context, request *storage.Notifier) (*storage.Notifier, error)
	TestNotifier(ctx context.Context, request *storage.Notifier) (*v1.Empty, error)
	DeleteNotifier(ctx context.Context, request *v1.DeleteNotifierRequest) (*v1.Empty, error)
}

// New returns a new Service instance using the given DataStore.
func New(storage datastore.DataStore,
	processor processor.Processor,
	buildTimePolicies detection.PolicySet,
	deployTimePolicies detection.PolicySet,
	runTimePolicies detection.PolicySet) Service {
	return &serviceImpl{
		storage:            storage,
		processor:          processor,
		buildTimePolicies:  buildTimePolicies,
		deployTimePolicies: deployTimePolicies,
		runTimePolicies:    runTimePolicies,
	}
}
