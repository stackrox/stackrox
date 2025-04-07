package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/quay/zlog"
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
	opts = append(opts, indexer.InsecureSkipTLSVerify(req.GetContainerImage().GetInsecureSkipTlsVerify()))
	// Create index report.
	zlog.Info(ctx).
		Str("image_url", req.GetContainerImage().GetUrl()).
		Bool("has_auth", hasAuth).
		Bool("insecure_skip_tls_verify", req.GetContainerImage().GetInsecureSkipTlsVerify()).
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
	if !clairReport.Success {
		return nil, fmt.Errorf("internal error: create index report failed in state %q: %s", clairReport.State, clairReport.Err)
	}
	indexReport, err := mappers.ToProtoV4IndexReport(clairReport)
	if err != nil {
		zlog.Error(ctx).Err(err).Msg("internal error: converting to v4.IndexReport")
		return nil, err
	}
	indexReport.HashId = req.GetHashId()
	return indexReport, nil
}

func (s *indexerService) GetIndexReport(ctx context.Context, req *v4.GetIndexReportRequest) (*v4.IndexReport, error) {
	ctx = zlog.ContextWithValues(ctx,
		"component", "scanner/service/indexer.GetIndexReport",
		"hash_id", req.GetHashId(),
	)
	zlog.Info(ctx).Msg("getting index report for container image")
	ir, err := s.getIndexReport(ctx, req)
	switch {
	case errors.Is(err, errox.NotFound):
		zlog.Warn(ctx).Err(err).Send()
	case err != nil:
		zlog.Error(ctx).Err(err).Msg("internal error: converting to v4.IndexReport")
	}
	return ir, err
}

func (s *indexerService) getIndexReport(ctx context.Context, req *v4.GetIndexReportRequest) (*v4.IndexReport, error) {
	ccIR, err := getClairIndexReport(ctx, s.indexer, req.GetHashId())
	if err != nil {
		return nil, err
	}
	v4IR, err := mappers.ToProtoV4IndexReport(ccIR)
	if err != nil {
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

	ir, err := s.getIndexReport(ctx, &v4.GetIndexReportRequest{
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
	_, err := getClairIndexReport(ctx, s.indexer, req.GetHashId())
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
