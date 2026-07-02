package services

import (
	"context"
	"log/slog"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stackrox/rox/clair-adapter/indexer"
	"github.com/stackrox/rox/clair-adapter/mappers"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/grpc/authz/or"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var indexerAuth = perrpc.FromMap(map[authz.Authorizer][]string{
	or.Or(idcheck.CentralOnly(), idcheck.SensorsOnly(), idcheck.ScannerV4MatcherOnly()): {
		v4.Indexer_GetIndexReport_FullMethodName,
		v4.Indexer_HasIndexReport_FullMethodName,
	},
	or.Or(idcheck.CentralOnly(), idcheck.SensorsOnly()): {
		v4.Indexer_CreateIndexReport_FullMethodName,
		v4.Indexer_GetOrCreateIndexReport_FullMethodName,
	},
	or.Or(idcheck.CentralOnly()): {
		v4.Indexer_StoreIndexReport_FullMethodName,
	},
	or.Or(idcheck.ScannerV4MatcherOnly()): {
		v4.Indexer_GetRepositoryToCPEMapping_FullMethodName,
	},
})

// indexerService implements the Scanner V4 IndexerServer interface.
type indexerService struct {
	v4.UnimplementedIndexerServer
	idx indexer.Indexer
}

// NewIndexerService creates a new indexerService with the given indexer.
func NewIndexerService(idx indexer.Indexer) *indexerService {
	return &indexerService{
		idx: idx,
	}
}

// CreateIndexReport creates an index report for the specified container image.
func (s *indexerService) CreateIndexReport(ctx context.Context, req *v4.CreateIndexReportRequest) (*v4.IndexReport, error) {
	slog.InfoContext(ctx, "CreateIndexReport called", "hash_id", req.GetHashId())

	img := req.GetContainerImage()
	if img == nil {
		slog.ErrorContext(ctx, "CreateIndexReport failed: container image is required")
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
		slog.ErrorContext(ctx, "Failed to index container image", "error", err, "hash_id", req.GetHashId())
		return nil, status.Errorf(codes.Internal, "failed to index container image: %v", err)
	}

	// Convert to proto format
	protoReport, err := mappers.ToProtoIndexReport(ir)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to convert index report", "error", err, "hash_id", req.GetHashId())
		return nil, status.Errorf(codes.Internal, "failed to convert index report: %v", err)
	}

	// Set the hash ID from the request
	protoReport.HashId = req.GetHashId()

	slog.InfoContext(ctx, "CreateIndexReport completed", "hash_id", req.GetHashId())
	return protoReport, nil
}

// GetIndexReport retrieves an existing index report by manifest hash.
func (s *indexerService) GetIndexReport(ctx context.Context, req *v4.GetIndexReportRequest) (*v4.IndexReport, error) {
	slog.InfoContext(ctx, "GetIndexReport called", "hash_id", req.GetHashId())

	ir, found, err := s.idx.GetIndexReport(ctx, req.GetHashId())
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get index report", "error", err, "hash_id", req.GetHashId())
		return nil, status.Errorf(codes.Internal, "failed to get index report: %v", err)
	}
	if !found {
		slog.InfoContext(ctx, "Index report not found", "hash_id", req.GetHashId())
		return nil, status.Error(codes.NotFound, "index report not found")
	}

	// Convert to proto format
	protoReport, err := mappers.ToProtoIndexReport(ir)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to convert index report", "error", err, "hash_id", req.GetHashId())
		return nil, status.Errorf(codes.Internal, "failed to convert index report: %v", err)
	}

	// Set the hash ID from the request
	protoReport.HashId = req.GetHashId()

	slog.InfoContext(ctx, "GetIndexReport completed", "hash_id", req.GetHashId())
	return protoReport, nil
}

// GetOrCreateIndexReport retrieves an existing index report or creates a new one.
func (s *indexerService) GetOrCreateIndexReport(ctx context.Context, req *v4.GetOrCreateIndexReportRequest) (*v4.IndexReport, error) {
	slog.InfoContext(ctx, "GetOrCreateIndexReport called", "hash_id", req.GetHashId())

	// Try to get existing report first
	getReq := &v4.GetIndexReportRequest{
		HashId: req.GetHashId(),
	}

	report, err := s.GetIndexReport(ctx, getReq)
	if err == nil {
		// Found existing report
		slog.InfoContext(ctx, "GetOrCreateIndexReport completed (found existing)", "hash_id", req.GetHashId())
		return report, nil
	}

	// Check if error was NotFound
	st := status.Convert(err)
	if st.Code() != codes.NotFound {
		// Some other error occurred
		slog.ErrorContext(ctx, "GetOrCreateIndexReport failed", "error", err, "hash_id", req.GetHashId())
		return nil, err
	}

	// Report not found, create it
	slog.InfoContext(ctx, "Index report not found, creating new one", "hash_id", req.GetHashId())
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
func (s *indexerService) HasIndexReport(ctx context.Context, req *v4.HasIndexReportRequest) (*v4.HasIndexReportResponse, error) {
	slog.InfoContext(ctx, "HasIndexReport called", "hash_id", req.GetHashId())

	exists, err := s.idx.HasIndexReport(ctx, req.GetHashId())
	if err != nil {
		slog.ErrorContext(ctx, "Failed to check index report", "error", err, "hash_id", req.GetHashId())
		return nil, status.Errorf(codes.Internal, "failed to check index report: %v", err)
	}

	slog.InfoContext(ctx, "HasIndexReport completed", "hash_id", req.GetHashId(), "exists", exists)
	return &v4.HasIndexReportResponse{
		Exists: exists,
	}, nil
}

// StoreIndexReport is not implemented.
func (s *indexerService) StoreIndexReport(ctx context.Context, req *v4.StoreIndexReportRequest) (*v4.StoreIndexReportResponse, error) {
	slog.InfoContext(ctx, "StoreIndexReport called (not implemented)")
	return nil, status.Error(codes.Unimplemented, "StoreIndexReport is not implemented")
}

// GetRepositoryToCPEMapping is not implemented.
func (s *indexerService) GetRepositoryToCPEMapping(ctx context.Context, req *v4.GetRepositoryToCPEMappingRequest) (*v4.GetRepositoryToCPEMappingResponse, error) {
	slog.InfoContext(ctx, "GetRepositoryToCPEMapping called (not implemented)")
	return nil, status.Error(codes.Unimplemented, "GetRepositoryToCPEMapping is not implemented")
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *indexerService) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, indexerAuth.Authorized(ctx, fullMethodName)
}

// RegisterServiceServer registers the indexerService with the gRPC server.
func (s *indexerService) RegisterServiceServer(grpcServer *grpc.Server) {
	v4.RegisterIndexerServer(grpcServer, s)
}

// RegisterServiceHandler is a no-op for indexerService (no HTTP gateway needed).
func (s *indexerService) RegisterServiceHandler(_ context.Context, _ *runtime.ServeMux, _ *grpc.ClientConn) error {
	return nil
}
