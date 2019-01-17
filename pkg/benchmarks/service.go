package benchmarks

import (
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
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

// RegisterServiceHandler implements the APIService interface, but the agent does not accept calls over the gRPC gateway
func (s *BenchmarkResultsService) RegisterServiceHandler(context.Context, *runtime.ServeMux, *grpc.ClientConn) error {
	return nil
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *BenchmarkResultsService) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, idcheck.BenchmarkOnly().Authorized(ctx, fullMethodName)
}

// PostBenchmarkResult inserts a new benchmark result into the system
func (s *BenchmarkResultsService) PostBenchmarkResult(ctx context.Context, request *storage.BenchmarkResult) (*v1.Empty, error) {
	if request == nil {
		return &v1.Empty{}, status.Errorf(codes.InvalidArgument, "Request object must be non-nil")
	}
	s.relayer.Accept(request)
	return &v1.Empty{}, nil
}
