package service

import (
	"context"

	"github.com/stackrox/rox/central/detection"
	"github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/notifier/processor"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/integrationhealth"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Service provides the interface to the microservice that serves alert data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.NotifierServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(storage datastore.DataStore,
	processor processor.Processor,
	buildTimePolicies detection.PolicySet,
	deployTimePolicies detection.PolicySet,
	runTimePolicies detection.PolicySet,
	reporter integrationhealth.Reporter) Service {
	return &serviceImpl{
		storage:            storage,
		processor:          processor,
		buildTimePolicies:  buildTimePolicies,
		deployTimePolicies: deployTimePolicies,
		runTimePolicies:    runTimePolicies,
		reporter:           reporter,
	}
}
