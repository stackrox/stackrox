package service

import (
	"context"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
)

// NewPingService returns the PingService API.
func NewPingService() *PingService {
	return &PingService{}
}

// PingService manages the simple Ping service.
type PingService struct {
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *PingService) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterPingServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *PingService) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterPingServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// Ping implements v1.PingServiceServer, and it always returns a v1.PongMessage object.
func (s *PingService) Ping(context.Context, *empty.Empty) (*v1.PongMessage, error) {
	result := &v1.PongMessage{
		Status: "ok",
	}
	return result, nil
}
