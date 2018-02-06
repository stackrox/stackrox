package service

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/auth"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewBenchmarkScansService returns the BenchmarkScansService API.
func NewBenchmarkScansService(storage db.Storage) *BenchmarkScansService {
	return &BenchmarkScansService{
		scanStore: storage,
	}
}

// BenchmarkScansService is the struct that manages the benchmark API
type BenchmarkScansService struct {
	scanStore db.BenchmarkScansStorage
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *BenchmarkScansService) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterBenchmarkScanServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *BenchmarkScansService) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterBenchmarkScanServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// PostBenchmarkScan inserts a scan into the database
func (s *BenchmarkScansService) PostBenchmarkScan(ctx context.Context, scan *v1.BenchmarkScanMetadata) (*empty.Empty, error) {
	identity, err := auth.FromContext(ctx)
	if err != nil || identity.TLS.Name.ServiceType != v1.ServiceType_SENSOR_SERVICE {
		return nil, status.Error(codes.Unauthenticated, "only sensors are allowed")
	}
	return &empty.Empty{}, s.scanStore.AddScan(scan)
}

// ListBenchmarkScans lists all of the scans that match the request parameters
func (s *BenchmarkScansService) ListBenchmarkScans(ctx context.Context, request *v1.ListBenchmarkScansRequest) (*v1.ListBenchmarkScansResponse, error) {
	metadata, err := s.scanStore.ListBenchmarkScans(request)
	if err != nil {
		return nil, err
	}
	return &v1.ListBenchmarkScansResponse{
		ScanMetadata: metadata,
	}, nil
}

// GetBenchmarkScan retrieves a specific benchmark scan
func (s *BenchmarkScansService) GetBenchmarkScan(ctx context.Context, request *v1.GetBenchmarkScanRequest) (*v1.BenchmarkScan, error) {
	if request.GetScanId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Scan ID must be defined when retrieving a scan")
	}
	scan, exists, err := s.scanStore.GetBenchmarkScan(request)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Could not find scan id %v", request.GetScanId()))
	}
	return scan, nil
}
