package service

import (
	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/apollo/image_processor"
	"bitbucket.org/stack-rox/apollo/apollo/scanners"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/secrets"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewScannerService returns the ScannerService API.
func NewScannerService(storage db.ScannerStorage, processor *imageprocessor.ImageProcessor) *ScannerService {
	return &ScannerService{
		storage:   storage,
		processor: processor,
	}
}

// ScannerService is the struct that manages the Scanner API
type ScannerService struct {
	storage   db.ScannerStorage
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

// GetScanners retrieves all registries that matches the request filters
func (s *ScannerService) GetScanners(ctx context.Context, request *v1.GetScannersRequest) (*v1.GetScannersResponse, error) {
	scannersWithSecrets, err := s.storage.GetScanners(request)
	if err != nil {
		return nil, err
	}
	scannersWithoutSecrets := make([]*v1.Scanner, 0, len(scannersWithSecrets))
	for _, scannerWithSecret := range scannersWithSecrets {
		scrubbedConfig := secrets.ScrubSecrets(scannerWithSecret.Config)
		scannersWithoutSecrets = append(scannersWithoutSecrets, &v1.Scanner{
			Name:     scannerWithSecret.Name,
			Endpoint: scannerWithSecret.Endpoint,
			Config:   scrubbedConfig,
		})
	}
	return &v1.GetScannersResponse{Scanners: scannersWithoutSecrets}, nil
}

// PostScanner inserts a new Scanner into the system
func (s *ScannerService) PostScanner(ctx context.Context, request *v1.Scanner) (*empty.Empty, error) {
	scanner, err := scanners.CreateScanner(request)
	if err != nil {
		return nil, err
	}
	if err := s.storage.AddScanner(request); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	s.processor.UpdateScanner(scanner)
	return &empty.Empty{}, nil
}

// PutScanner updates a scanner in the system
func (s *ScannerService) PutScanner(ctx context.Context, request *v1.Scanner) (*empty.Empty, error) {
	scanner, err := scanners.CreateScanner(request)
	if err != nil {
		return nil, err
	}
	if err := s.storage.UpdateScanner(request); err != nil {
		return nil, err
	}
	s.processor.UpdateScanner(scanner)
	return &empty.Empty{}, nil
}
