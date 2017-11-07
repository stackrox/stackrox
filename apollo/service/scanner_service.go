package service

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/apollo/scanners"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewScannerService returns the ScannerService API.
func NewScannerService(storage db.Storage) *ScannerService {
	return &ScannerService{
		storage: storage,
	}
}

// ScannerService is the struct that manages the Scanner API
type ScannerService struct {
	storage db.Storage
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *ScannerService) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterScannerServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *ScannerService) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterScannerServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// PostScanner inserts a new Scanner into the system
func (s *ScannerService) PostScanner(ctx context.Context, request *v1.Scanner) (*v1.Scanner, error) {
	creator, exists := scanners.Registry[request.Name]
	if !exists {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Registry with type %v does not exist", request.Name))
	}
	scanner, err := creator(request.Endpoint, request.Config)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	s.storage.AddScanner(request.Name, scanner)
	return request, nil
}

// GetScanners retrieves all registries that matches the request filters
func (s *ScannerService) GetScanners(ctx context.Context, request *v1.GetScannersRequest) (*v1.GetScannersResponse, error) {
	scannersWithSecrets := s.storage.GetScanners()
	scannersWithoutSecrets := make([]*v1.Scanner, 0, len(scannersWithSecrets))
	for name, scannerWithSecret := range scannersWithSecrets {
		config := scannerWithSecret.Config()
		for _, secretKey := range secretKeys {
			delete(config, secretKey)
		}
		scannersWithoutSecrets = append(scannersWithoutSecrets, &v1.Scanner{
			Name:     name,
			Endpoint: scannerWithSecret.Endpoint(),
			Config:   config,
		})
	}
	return &v1.GetScannersResponse{Scanners: scannersWithoutSecrets}, nil
}
