package benchmarks

import (
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewBenchmarkResultsService returns the BenchmarkResultsService API for Sensors.
func NewBenchmarkResultsService(relayer Relayer) *BenchmarkResultsService {
	return &BenchmarkResultsService{
		relayer: relayer,
	}
}

// BenchmarkResultsService is the struct that manages the benchmark results API
type BenchmarkResultsService struct {
	relayer Relayer
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *BenchmarkResultsService) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterBenchmarkResultsServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint implements the APIService interface, but the agent does not accept calls over the gRPC gateway
func (s *BenchmarkResultsService) RegisterServiceHandlerFromEndpoint(context.Context, *runtime.ServeMux, string, []grpc.DialOption) error {
	return nil
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *BenchmarkResultsService) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	// TODO(cg): AP-157: Provide credentials to the benchmark service and verify them here.
	return ctx, allow.Anonymous().Authorized(ctx, fullMethodName)
}

// PostBenchmarkResult inserts a new benchmark result into the system
func (s *BenchmarkResultsService) PostBenchmarkResult(ctx context.Context, request *v1.BenchmarkResult) (*empty.Empty, error) {
	if request == nil {
		return &empty.Empty{}, status.Errorf(codes.InvalidArgument, "Request object must be non-nil")
	}
	s.relayer.Accept(request)
	return &empty.Empty{}, nil
}
