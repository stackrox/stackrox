package services

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/clair-adapter/indexer"
	"github.com/stackrox/rox/clair-adapter/mappers"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// IndexerService implements the Scanner V4 IndexerServer interface.
type IndexerService struct {
	v4.UnimplementedIndexerServer
	idx indexer.Indexer
}

// NewIndexerService creates a new IndexerService with the given indexer.
func NewIndexerService(idx indexer.Indexer) *IndexerService {
	return &IndexerService{
		idx: idx,
	}
}

// CreateIndexReport creates an index report for the specified container image.
func (s *IndexerService) CreateIndexReport(ctx context.Context, req *v4.CreateIndexReportRequest) (*v4.IndexReport, error) {
	img := req.GetContainerImage()
	if img == nil {
		return nil, status.Error(codes.InvalidArgument, "container image is required")
	}

	// Build indexer options from image credentials
	var opts []indexer.Option
	if img.GetUsername() != "" || img.GetPassword() != "" {
		opts = append(opts, indexer.WithBasicAuth(img.GetUsername(), img.GetPassword()))
	}
	if img.GetInsecureSkipTlsVerify() {
		opts = append(opts, indexer.WithInsecureSkipTLSVerify(img.GetInsecureSkipTlsVerify()))
	}

	// Index the container image
	ir, err := s.idx.IndexContainerImage(ctx, req.GetHashId(), img.GetUrl(), opts...)
	if err != nil {
		return nil, status.Error(codes.Internal, errors.Wrap(err, "failed to index container image").Error())
	}

	// Convert to proto format
	protoReport, err := mappers.ToProtoIndexReport(ir)
	if err != nil {
		return nil, status.Error(codes.Internal, errors.Wrap(err, "failed to convert index report").Error())
	}

	// Set the hash ID from the request
	protoReport.HashId = req.GetHashId()

	return protoReport, nil
}

// GetIndexReport retrieves an existing index report by manifest hash.
func (s *IndexerService) GetIndexReport(ctx context.Context, req *v4.GetIndexReportRequest) (*v4.IndexReport, error) {
	ir, found, err := s.idx.GetIndexReport(ctx, req.GetHashId())
	if err != nil {
		return nil, status.Error(codes.Internal, errors.Wrap(err, "failed to get index report").Error())
	}
	if !found {
		return nil, status.Error(codes.NotFound, "index report not found")
	}

	// Convert to proto format
	protoReport, err := mappers.ToProtoIndexReport(ir)
	if err != nil {
		return nil, status.Error(codes.Internal, errors.Wrap(err, "failed to convert index report").Error())
	}

	// Set the hash ID from the request
	protoReport.HashId = req.GetHashId()

	return protoReport, nil
}

// GetOrCreateIndexReport retrieves an existing index report or creates a new one.
func (s *IndexerService) GetOrCreateIndexReport(ctx context.Context, req *v4.GetOrCreateIndexReportRequest) (*v4.IndexReport, error) {
	// Try to get existing report first
	getReq := &v4.GetIndexReportRequest{
		HashId: req.GetHashId(),
	}

	report, err := s.GetIndexReport(ctx, getReq)
	if err == nil {
		// Found existing report
		return report, nil
	}

	// Check if error was NotFound
	st := status.Convert(err)
	if st.Code() != codes.NotFound {
		// Some other error occurred
		return nil, err
	}

	// Report not found, create it
	createReq := &v4.CreateIndexReportRequest{
		HashId: req.GetHashId(),
	}

	// Copy the resource locator
	if img := req.GetContainerImage(); img != nil {
		createReq.ResourceLocator = &v4.CreateIndexReportRequest_ContainerImage{
			ContainerImage: img,
		}
	}

	return s.CreateIndexReport(ctx, createReq)
}

// HasIndexReport checks if an index report exists for the given manifest hash.
func (s *IndexerService) HasIndexReport(ctx context.Context, req *v4.HasIndexReportRequest) (*v4.HasIndexReportResponse, error) {
	exists, err := s.idx.HasIndexReport(ctx, req.GetHashId())
	if err != nil {
		return nil, status.Error(codes.Internal, errors.Wrap(err, "failed to check index report").Error())
	}

	return &v4.HasIndexReportResponse{
		Exists: exists,
	}, nil
}

// StoreIndexReport is not implemented.
func (s *IndexerService) StoreIndexReport(ctx context.Context, req *v4.StoreIndexReportRequest) (*v4.IndexReport, error) {
	return nil, status.Error(codes.Unimplemented, "StoreIndexReport is not implemented")
}

// GetRepositoryToCPEMapping is not implemented.
func (s *IndexerService) GetRepositoryToCPEMapping(ctx context.Context, req *v4.GetRepositoryToCPEMappingRequest) (*v4.GetRepositoryToCPEMappingResponse, error) {
	return nil, status.Error(codes.Unimplemented, "GetRepositoryToCPEMapping is not implemented")
}
