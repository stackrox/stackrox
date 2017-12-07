package service

import (
	"bitbucket.org/stack-rox/apollo/agent/swarm/benchmarks"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewBenchmarkRelayService returns the BenchmarkRelayService API.
func NewBenchmarkRelayService(relayer benchmarks.Relayer) *BenchmarkRelayService {
	return &BenchmarkRelayService{
		relayer: relayer,
	}
}

// BenchmarkRelayService is the struct that manages the benchmark API
type BenchmarkRelayService struct {
	relayer benchmarks.Relayer
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *BenchmarkRelayService) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterBenchmarkRelayServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *BenchmarkRelayService) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterBenchmarkRelayServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// PostBenchmarkResult inserts a new benchmark result into the system
func (s *BenchmarkRelayService) PostBenchmarkResult(ctx context.Context, request *v1.BenchmarkPayload) (*empty.Empty, error) {
	if request == nil {
		return &empty.Empty{}, status.Errorf(codes.InvalidArgument, "Request object must be non-nil")
	}
	s.relayer.Accept(request)
	return &empty.Empty{}, nil
}
