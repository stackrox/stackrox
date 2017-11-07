package service

import (
	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/apollo/image_processor"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewScannerService returns the ScannerService API.
func NewScannerService(storage db.Storage, processor *imageprocessor.ImageProcessor) *ScannerService {
	return &ScannerService{
		storage:   storage,
		processor: processor,
	}
}

// ScannerService is the struct that manages the Scanner API
type ScannerService struct {
	storage   db.Storage
	processor *imageprocessor.ImageProcessor
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *ScannerService) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterScannerServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *ScannerService) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterScannerServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// GetScanners retrieves scanners based on the filters passed by the request
func (s *ScannerService) GetScanners(ctx context.Context, request *v1.GetScannersRequest) (*v1.GetScannersResponse, error) {
	return &v1.GetScannersResponse{}, status.Error(codes.Unimplemented, "Not implemented")
}

// PostScanner inserts a new Scanner into the system
func (s *ScannerService) PostScanner(ctx context.Context, request *v1.Scanner) (*v1.Scanner, error) {
	return request, status.Error(codes.Unimplemented, "Not implemented")
}
