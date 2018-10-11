package sensornetworkflow

import (
	"github.com/stackrox/rox/central/networkflow/store"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/timestamp"
	"golang.org/x/net/context"
)

var (
	log = logging.LoggerForModule()
)

// Service is the GRPC service interface that provides the entry point for processing deployment events.
type Service interface {
	grpc.APIService
	central.NetworkFlowServiceServer

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
}

// New returns a new instance of service.
func New(store store.ClusterStore) Service {
	return &serviceImpl{
		clusterStore: store,
		lastUpdateTS: make(map[string]timestamp.MicroTS),
	}
}
