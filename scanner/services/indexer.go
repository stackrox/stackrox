package services

import (
	"context"
	"errors"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/quay/claircore"
	"github.com/quay/zlog"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/grpc/authz/or"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/scanner/indexer"
	"github.com/stackrox/rox/scanner/mappers"
	"github.com/stackrox/rox/scanner/services/validators"
	"google.golang.org/grpc"
)

var indexerAuth = perrpc.FromMap(map[authz.Authorizer][]string{
	or.Or(idcheck.CentralOnly(), idcheck.SensorsOnly(), idcheck.ScannerV4MatcherOnly()): {
		"/scanner.v4.Indexer/GetIndexReport",
		"/scanner.v4.Indexer/HasIndexReport",
	},
	or.Or(idcheck.CentralOnly(), idcheck.SensorsOnly()): {
		// Matcher should never attempt to create an index report.
		"/scanner.v4.Indexer/CreateIndexReport",
		"/scanner.v4.Indexer/GetOrCreateIndexReport",
	},
})

type indexerService struct {
	v4.UnimplementedIndexerServer
	indexer indexer.Indexer
}

// NewIndexerService creates a new indexer service.
func NewIndexerService(indexer indexer.Indexer) *indexerService {
	return &indexerService{
		indexer: indexer,
	}
}

func (s *indexerService) CreateIndexReport(ctx context.Context, req *v4.CreateIndexReportRequest) (*v4.IndexReport, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "scanner/service/indexer.CreateIndexReport")
	// TODO We currently only support container images, hence we assume the resource
	//      is of that type. When introducing nodes and other resources, this should
	//      evolve.
	resourceType := "containerimage"
	if err := validators.ValidateContainerImageRequest(req); err != nil {
		return nil, err
	}
	ctx = zlog.ContextWithValues(ctx, "resource_type", resourceType, "hash_id", req.GetHashId())

	// Setup authentication.
	var opts []indexer.Option
	hasAuth := req.GetContainerImage().GetUsername() != ""
	if hasAuth {
		opts = append(opts, indexer.WithAuth(&authn.Basic{
			Username: req.GetContainerImage().GetUsername(),
			Password: req.GetContainerImage().GetPassword(),
		}))
	}

	// Create index report.
	zlog.Info(ctx).
		Str("image_url", req.GetContainerImage().GetUrl()).
		Bool("has_auth", hasAuth).
		Msg("creating index report for container image")
	clairReport, err := s.indexer.IndexContainerImage(
		ctx,
		req.GetHashId(),
		req.GetContainerImage().GetUrl(),
		opts...)
	if err != nil {
		zlog.Error(ctx).Err(err).Send()
		return nil, err
	}
	indexReport, err := mappers.ToProtoV4IndexReport(clairReport)
	if err != nil {
		zlog.Error(ctx).Err(err).Msg("internal error: converting to v4.IndexReport")
		return nil, err
	}
	indexReport.HashId = req.GetHashId()
	// TODO Define behavior for indexReport.Err != "".
	return indexReport, nil
}

func (s *indexerService) GetIndexReport(ctx context.Context, req *v4.GetIndexReportRequest) (*v4.IndexReport, error) {
	ctx = zlog.ContextWithValues(ctx,
		"component", "scanner/service/indexer.GetIndexReport",
		"hash_id", req.GetHashId(),
	)
	// Create index report.
	zlog.Info(ctx).Msg("getting index report for container image")
	ccIR, err := s.getClairIndexReport(ctx, req.GetHashId())
	if err != nil {
		zlog.Error(ctx).Err(err).Send()
		return nil, err
	}
	v4IR, err := mappers.ToProtoV4IndexReport(ccIR)
	if err != nil {
		zlog.Error(ctx).Err(err).Msg("internal error: converting to v4.IndexReport")
		return nil, err
	}
	v4IR.HashId = req.GetHashId()
	return v4IR, nil
}

func (s *indexerService) GetOrCreateIndexReport(ctx context.Context, req *v4.GetOrCreateIndexReportRequest) (*v4.IndexReport, error) {
	ctx = zlog.ContextWithValues(ctx,
		"component", "scanner/service/indexer.GetOrCreateIndexReport",
		"hash_id", req.GetHashId(),
	)

	ir, err := s.GetIndexReport(ctx, &v4.GetIndexReportRequest{
		HashId: req.GetHashId(),
	})
	switch {
	case errors.Is(err, nil):
		return ir, nil
	case errors.Is(err, errox.NotFound):
		// Not found, log and go create.
		zlog.Debug(ctx).Err(err).Msg("index report not found")
	default:
		return nil, err
	}

	// TODO We currently only support container images, hence we assume the resource
	//      is of that type. When introducing nodes and other resources, this should
	//      evolve.
	return s.CreateIndexReport(ctx, &v4.CreateIndexReportRequest{
		HashId: req.GetHashId(),
		ResourceLocator: &v4.CreateIndexReportRequest_ContainerImage{
			ContainerImage: req.GetContainerImage(),
		},
	})
}

func (s *indexerService) HasIndexReport(ctx context.Context, req *v4.HasIndexReportRequest) (*v4.HasIndexReportResponse, error) {
	ctx = zlog.ContextWithValues(ctx,
		"component", "scanner/service/indexer.HasIndexReport",
		"hash_id", req.GetHashId(),
	)
	_, err := s.getClairIndexReport(ctx, req.GetHashId())
	var exists bool
	switch {
	case errors.Is(err, nil):
		exists = true
	case errors.Is(err, errox.NotFound):
		exists = false
	default:
		zlog.Error(ctx).Err(err).Msg("failed retrieve index report")
		return nil, err
	}
	return &v4.HasIndexReportResponse{Exists: exists}, nil
}

// getClairIndexReport query and return a claircore index report, return a "not
// found" error when the report does not exist or if it is not successful.
func (s *indexerService) getClairIndexReport(ctx context.Context, hashID string) (*claircore.IndexReport, error) {
	ir, found, err := s.indexer.GetIndexReport(ctx, hashID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errox.NotFound.Newf("report %q not found", hashID)
	}
	if !ir.Success {
		return nil, errox.NotFound.Newf("unsuccessful state %q: %s", ir.State, ir.Err)
	}
	return ir, nil
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *indexerService) RegisterServiceServer(grpcServer *grpc.Server) {
	v4.RegisterIndexerServer(grpcServer, s)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *indexerService) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	auth := indexerAuth
	// If this a dev build, allow anonymous traffic for testing purposes.
	if !buildinfo.ReleaseBuild {
		auth = allow.Anonymous()
	}
	return ctx, auth.Authorized(ctx, fullMethodName)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *indexerService) RegisterServiceHandler(_ context.Context, _ *runtime.ServeMux, _ *grpc.ClientConn) error {
	// Currently we do not set up gRPC gateway for the matcher.
	return nil
}
