package service

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/central/detection"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authn"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/or"
	"bitbucket.org/stack-rox/apollo/pkg/scanners"
	"bitbucket.org/stack-rox/apollo/pkg/secrets"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewScannerService returns the ScannerService API.
func NewScannerService(storage db.ScannerStorage, detector *detection.Detector) *ScannerService {
	return &ScannerService{
		storage:  storage,
		detector: detector,
	}
}

// ScannerService is the struct that manages the Scanner API
type ScannerService struct {
	storage  db.ScannerStorage
	detector *detection.Detector
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *ScannerService) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterScannerServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *ScannerService) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterScannerServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *ScannerService) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, returnErrorCode(or.SensorOrUser().Authorized(ctx))
}

// GetScanner retrieves the scanner based on the id passed
func (s *ScannerService) GetScanner(ctx context.Context, request *v1.ResourceByID) (*v1.Scanner, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Scanner id must be provided")
	}
	scanner, exists, err := s.storage.GetScanner(request.GetId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Scanner %v not found", request.GetId()))
	}
	return scanner, nil
}

// GetScanners retrieves all scanners that matches the request filters
func (s *ScannerService) GetScanners(ctx context.Context, request *v1.GetScannersRequest) (*v1.GetScannersResponse, error) {
	scanners, err := s.storage.GetScanners(request)
	if err != nil {
		return nil, err
	}
	identity, err := authn.FromTLSContext(ctx)
	switch {
	case err == authn.ErrNoContext:
		log.Debugf("No authentication context provided")
	case err != nil:
		log.Warnf("Error getting client identity: %s", err)
	case err == nil && identity.Name.ServiceType == v1.ServiceType_SENSOR_SERVICE:
		return &v1.GetScannersResponse{Scanners: scanners}, nil
	}

	// Remove secrets for other API accessors.
	for _, s := range scanners {
		s.Config = secrets.ScrubSecrets(s.Config)
	}
	return &v1.GetScannersResponse{Scanners: scanners}, nil
}

// PostScanner inserts a new Scanner into the system
func (s *ScannerService) PostScanner(ctx context.Context, request *v1.Scanner) (*v1.Scanner, error) {
	if request.GetId() != "" {
		return nil, status.Error(codes.InvalidArgument, "Id field should be empty when posting a new scanner")
	}
	scanner, err := scanners.CreateScanner(request)
	if err != nil {
		return nil, err
	}
	id, err := s.storage.AddScanner(request)
	if err != nil {
		return nil, err
	}
	request.Id = id
	s.detector.UpdateScanner(scanner)
	return request, nil
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
	s.detector.UpdateScanner(scanner)
	return &empty.Empty{}, nil
}

// DeleteScanner deletes a scanner from the system
func (s *ScannerService) DeleteScanner(ctx context.Context, request *v1.ResourceByID) (*empty.Empty, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Scanner id must be provided")
	}
	if err := s.storage.RemoveScanner(request.GetId()); err != nil {
		return nil, returnErrorCode(err)
	}
	s.detector.RemoveScanner(request.GetId())
	return &empty.Empty{}, nil
}
