package service

import (
	"context"

	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common/metrics"
	"github.com/stackrox/rox/sensor/common/networkflow/manager"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type serviceImpl struct {
	manager manager.Manager
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	sensor.RegisterNetworkConnectionInfoServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	// There is no grpc gateway handler for network connection info service
	return nil
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, idcheck.CollectorOnly().Authorized(ctx, fullMethodName)
}

// PushSignals handles the bidirectional gRPC stream with the collector
func (s *serviceImpl) PushNetworkConnectionInfo(stream sensor.NetworkConnectionInfoService_PushNetworkConnectionInfoServer) error {
	return s.receiveMessages(stream)
}

func (s *serviceImpl) receiveMessages(stream sensor.NetworkConnectionInfoService_PushNetworkConnectionInfoServer) error {
	var hostname string

	incomingMD := metautils.ExtractIncoming(stream.Context())
	hostname = incomingMD.Get("rox-collector-hostname")
	if hostname == "" {
		return status.Error(codes.Internal, "collector did not transmit a hostname in initial metadata")
	}

	if err := stream.SendHeader(metadata.MD{}); err != nil {
		return status.Errorf(codes.Internal, "error sending initial metadata: %v", err)
	}

	hostConnections, sequenceID := s.manager.RegisterCollector(hostname)

	for stream.Context().Err() == nil {
		msg, err := stream.Recv()
		if err != nil {
			log.Errorf("error dequeueing message: %s", err)
			return status.Errorf(codes.Internal, "error dequeueing message: %s", err)
		}

		networkInfoMsg := msg.GetInfo()
		networkInfoMsgTimestamp := timestamp.Now()

		if networkInfoMsg == nil {
			return status.Errorf(codes.Internal, "received unexpected message type %T from hostname %s", networkInfoMsg, hostname)
		}

		metrics.IncrementTotalNetworkFlowsReceivedCounter(env.ClusterID.Setting(), len(msg.GetInfo().GetUpdatedConnections()))
		if err := hostConnections.Process(networkInfoMsg, networkInfoMsgTimestamp, sequenceID); err != nil {
			return status.Errorf(codes.Internal, "could not process connections: %v", err)
		}
	}

	return stream.Context().Err()
}
