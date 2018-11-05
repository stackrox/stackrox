package sensornetworkflow

import (
	"fmt"
	"sync"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/networkflow/store"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/grpc/clusters"
	"github.com/stackrox/rox/pkg/timestamp"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type serviceImpl struct {
	clusterStore store.ClusterStore

	lastUpdateTSMutex sync.Mutex
	lastUpdateTS      map[string]timestamp.MicroTS
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	central.RegisterNetworkFlowServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return nil
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, idcheck.SensorsOnly().Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) PushNetworkFlows(stream central.NetworkFlowService_PushNetworkFlowsServer) error {
	err := s.receiveNetworkFlowUpdates(stream)
	return err
}

func (s *serviceImpl) receiveNetworkFlowUpdates(stream central.NetworkFlowService_PushNetworkFlowsServer) error {

	clusterID := clusters.IDFromContext(stream.Context())
	if clusterID == "" {
		return status.Errorf(codes.Internal, "unable to get cluster ID from sensor stream")
	}

	flowStore, err := s.clusterStore.CreateFlowStore(clusterID)
	if err != nil {
		return fmt.Errorf("could not get or create flow store for cluster %s: %v", clusterID, err)
	}
	updater := newFlowStoreUpdater(flowStore)

	if err := stream.SendHeader(metadata.MD{}); err != nil {
		return status.Errorf(codes.Internal, "sending initial metadata: %v", err)
	}

	for {
		update, err := stream.Recv()
		if err != nil {
			return fmt.Errorf("receiving message: %v", err)
		}
		if len(update.Updated) == 0 {
			return status.Errorf(codes.Internal, "received empty updated flows")
		}

		metrics.IncrementTotalNetworkFlowsReceivedCounter(clusterID, len(update.Updated))
		if err = updater.update(update.Updated, update.Time); err != nil {
			return status.Errorf(codes.Internal, err.Error())
		}
	}
}
