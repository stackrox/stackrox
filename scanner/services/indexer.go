package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/quay/claircore/toolkit/log"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/grpc/authz/or"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/scannerv4/mappers"
	"github.com/stackrox/rox/scanner/indexer"
	"github.com/stackrox/rox/scanner/services/validators"
	"google.golang.org/grpc"
)

var indexerAuth = perrpc.FromMap(map[authz.Authorizer][]string{
	or.Or(idcheck.CentralOnly(), idcheck.SensorsOnly(), idcheck.ScannerV4MatcherOnly()): {
		v4.Indexer_GetIndexReport_FullMethodName,
		v4.Indexer_HasIndexReport_FullMethodName,
	},
	or.Or(idcheck.CentralOnly(), idcheck.SensorsOnly()): {
		// Matcher should never attempt to create an index report.
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

type indexerService struct {
	v4.UnimplementedIndexerServer
	// indexer is used to retrieve index reports.
	indexer indexer.Indexer
	// anonymousAuthEnabled specifies if the service should allow for traffic from anonymous users.
	anonymousAuthEnabled bool
}

// NewIndexerService creates a new indexer service.
func NewIndexerService(indexer indexer.Indexer) *indexerService {
	return &indexerService{
		indexer:              indexer,
		anonymousAuthEnabled: env.ScannerV4AnonymousAuth.BooleanSetting(),
	}
}

func (s *indexerService) CreateIndexReport(ctx context.Context, req *v4.CreateIndexReportRequest) (*v4.IndexReport, error) {
	return s.createIndexReport(ctx, req)
}

// createIndexReport creates an Index Report for the given request.
// This function writes logs using the given context.
func (s *indexerService) createIndexReport(ctx context.Context, req *v4.CreateIndexReportRequest) (*v4.IndexReport, error) {
	slog.InfoContext(ctx, "creating index report for container image")

	// TODO We currently only support container images, hence we assume the resource
	//      is of that type. When introducing nodes and other resources, this should
	//      evolve.
	resourceType := "containerimage"
	if err := validators.ValidateContainerImageRequest(req); err != nil {
		slog.ErrorContext(ctx, "invalid request", "reason", err)
		return nil, err
	}
	ctx = log.With(ctx, "resource_type", resourceType, "hash_id", req.GetHashId())

	// Setup authentication.
	var opts []indexer.Option
	hasAuth := req.GetContainerImage().GetUsername() != ""
	if hasAuth {
		opts = append(opts, indexer.WithAuth(&authn.Basic{
			Username: req.GetContainerImage().GetUsername(),
			Password: req.GetContainerImage().GetPassword(),
		}))
	}
	opts = append(opts, indexer.InsecureSkipTLSVerify(req.GetContainerImage().GetInsecureSkipTlsVerify()))
	// Create index report.
	slog.InfoContext(ctx, "creating index report for container image",
		"image_url", req.GetContainerImage().GetUrl(),
		"has_auth", hasAuth,
		"insecure_skip_tls_verify", req.GetContainerImage().GetInsecureSkipTlsVerify())
	clairReport, err := s.indexer.IndexContainerImage(
		ctx,
		req.GetHashId(),
		req.GetContainerImage().GetUrl(),
		opts...)
	if err != nil {
		slog.ErrorContext(ctx, "indexing container image failed", "reason", err)
		return nil, err
	}
	if !clairReport.Success {
		return nil, fmt.Errorf("internal error: create index report failed in state %q: %s", clairReport.State, clairReport.Err)
	}
	indexReport, err := mappers.ToProtoV4IndexReport(clairReport)
	if err != nil {
		slog.ErrorContext(ctx, "internal error: converting to v4.IndexReport", "reason", err)
		return nil, err
	}
	indexReport.HashId = req.GetHashId()
	return indexReport, nil
}

func (s *indexerService) GetIndexReport(ctx context.Context, req *v4.GetIndexReportRequest) (*v4.IndexReport, error) {
	ctx = log.With(ctx, "hash_id", req.GetHashId())
	slog.InfoContext(ctx, "getting index report for container image")
	ir, err := s.getIndexReport(ctx, req.GetHashId(), req.GetIncludeExternal())
	switch {
	case errors.Is(err, errox.NotFound):
		slog.WarnContext(ctx, "index report not found", "reason", err)
	case err != nil:
		slog.ErrorContext(ctx, "internal error", "reason", err)
	}
	return ir, err
}

// getIndexReport fetches the Index Report for the resource with the given hashID.
// No logging is performed; however, callers of this method may be
// interested in logging the error. Returns errox.NotFound when the report does
// not exist.
func (s *indexerService) getIndexReport(ctx context.Context, hashID string, includeExternal bool) (*v4.IndexReport, error) {
	ccIR, err := getClairIndexReport(ctx, s.indexer, hashID, includeExternal)
	if err != nil {
		return nil, err
	}
	v4IR, err := mappers.ToProtoV4IndexReport(ccIR)
	if err != nil {
		return nil, err
	}
	v4IR.HashId = hashID
	return v4IR, nil
}

func (s *indexerService) GetOrCreateIndexReport(ctx context.Context, req *v4.GetOrCreateIndexReportRequest) (*v4.IndexReport, error) {
	ctx = log.With(ctx, "hash_id", req.GetHashId())

	slog.InfoContext(ctx, "getting index report for container image")
	ir, err := s.getIndexReport(ctx, req.GetHashId(), false)
	switch {
	case errors.Is(err, nil):
		return ir, nil
	case errors.Is(err, errox.NotFound):
		// Not found, log and go create.
		slog.DebugContext(ctx, "index report not found", "reason", err)
	default:
		slog.ErrorContext(ctx, "internal error", "reason", err)
		return nil, err
	}

	// TODO We currently only support container images, hence we assume the resource
	//      is of that type. When introducing nodes and other resources, this should
	//      evolve.
	return s.createIndexReport(ctx, &v4.CreateIndexReportRequest{
		HashId: req.GetHashId(),
		ResourceLocator: &v4.CreateIndexReportRequest_ContainerImage{
			ContainerImage: req.GetContainerImage(),
		},
	})
}

func (s *indexerService) HasIndexReport(ctx context.Context, req *v4.HasIndexReportRequest) (*v4.HasIndexReportResponse, error) {
	ctx = log.With(ctx, "hash_id", req.GetHashId())
	_, err := getClairIndexReport(ctx, s.indexer, req.GetHashId(), false)
	var exists bool
	switch {
	case errors.Is(err, nil):
		exists = true
	case errors.Is(err, errox.NotFound):
		exists = false
	default:
		slog.ErrorContext(ctx, "failed retrieve index report", "reason", err)
		return nil, err
	}
	return &v4.HasIndexReportResponse{Exists: exists}, nil
}

func (s *indexerService) StoreIndexReport(ctx context.Context, req *v4.StoreIndexReportRequest) (*v4.StoreIndexReportResponse, error) {
	ctx = log.With(ctx, "hash_id", req.GetHashId())

	resp := &v4.StoreIndexReportResponse{Status: "ERROR"}
	if req.GetContents() == nil {
		slog.DebugContext(ctx, "no contents, rejecting")
		return resp, errox.InvalidArgs.New("empty contents")
	}

	slog.InfoContext(ctx, "storing external index report")
	ir, err := parseIndexReport(req.GetContents())
	if err != nil {
		return resp, fmt.Errorf("parsing contents to index report: %w", err)
	}

	resp.Status, err = s.indexer.StoreIndexReport(ctx, req.GetHashId(), req.GetIndexerVersion(), ir)
	if err != nil {
		return resp, fmt.Errorf("storing external index report: %w", err)
	}

	return resp, nil
}

func (s *indexerService) GetRepositoryToCPEMapping(ctx context.Context, _ *v4.GetRepositoryToCPEMappingRequest) (*v4.GetRepositoryToCPEMappingResponse, error) {
	slog.InfoContext(ctx, "getting repository-to-CPE mapping")

	mf, err := s.indexer.GetRepositoryToCPEMapping(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get repository-to-CPE mapping", "reason", err)
		return nil, err
	}

	// Convert to proto format.
	result := make(map[string]*v4.RepositoryCPEInfo, len(mf.Data))
	for repo, info := range mf.Data {
		if len(info.CPEs) > 0 {
			result[repo] = &v4.RepositoryCPEInfo{Cpes: info.CPEs}
		}
	}

	return &v4.GetRepositoryToCPEMappingResponse{Mapping: result}, nil
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *indexerService) RegisterServiceServer(grpcServer *grpc.Server) {
	v4.RegisterIndexerServer(grpcServer, s)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *indexerService) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	auth := indexerAuth
	if s.anonymousAuthEnabled {
		auth = allow.Anonymous()
	}
	return ctx, auth.Authorized(ctx, fullMethodName)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *indexerService) RegisterServiceHandler(_ context.Context, _ *runtime.ServeMux, _ *grpc.ClientConn) error {
	// Currently we do not set up gRPC gateway for the matcher.
	return nil
}
