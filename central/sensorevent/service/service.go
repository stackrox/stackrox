package service

import (
	"github.com/stackrox/rox/central/sensorevent/service/pipeline"
	deploymentEventStore "github.com/stackrox/rox/central/sensorevent/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/logging"
	"golang.org/x/net/context"
)

var (
	log = logging.LoggerForModule()
)

// Service is the GRPC service interface that provides the entry point for processing deployment events.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	RecordEvent(stream v1.SensorEventService_RecordEventServer) error
}

// New returns a new instance of service.
func New(deploymentEvents deploymentEventStore.Store,
	pl pipeline.Pipeline) Service {
	return &serviceImpl{
		deploymentEvents: deploymentEvents,
		pl:               pl,
	}
}
