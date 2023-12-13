package services

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/quay/claircore"
	"github.com/quay/zlog"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/scanner/indexer"
	"github.com/stackrox/rox/scanner/mappers"
	"github.com/stackrox/rox/scanner/matcher"
	"github.com/stackrox/rox/scanner/services/validators"
	"google.golang.org/grpc"
)

// matcherService represents a vulnerability matcher gRPC service.
type matcherService struct {
	v4.UnimplementedMatcherServer
	// indexer is used to retrieve index reports.
	indexer indexer.ReportGetter
	// matcher is used to match vulnerabilities with index contents.
	matcher matcher.Matcher
	// disableEmptyContents allows the vulnerability matching API to reject requests with empty contents.
	disableEmptyContents bool
}

// NewMatcherService creates a new vulnerability matcher gRPC service, to enable
// empty content in enrich requests, pass a non-nil indexer.
func NewMatcherService(matcher matcher.Matcher, indexer indexer.ReportGetter) *matcherService {
	return &matcherService{
		matcher:              matcher,
		indexer:              indexer,
		disableEmptyContents: indexer == nil,
	}
}

func (s *matcherService) GetVulnerabilities(ctx context.Context, req *v4.GetVulnerabilitiesRequest) (*v4.VulnerabilityReport, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "scanner/service/matcher")
	if err := validators.ValidateGetVulnerabilitiesRequest(req); err != nil {
		return nil, errox.InvalidArgs.CausedBy(err)
	}
	// Get an index report to enrich: either using the indexer, or provided in the request.
	var ir *claircore.IndexReport
	var err error
	if req.GetContents() == nil {
		if s.disableEmptyContents {
			zlog.Debug(ctx).Msg("no contents, rejecting")
			return nil, errox.InvalidArgs.New("empty contents is disabled")
		}
		zlog.Debug(ctx).Msg("no contents, retrieving")
		ir, err = s.retrieveIndexReport(ctx, req.GetHashId())
	} else {
		zlog.Info(ctx).Msg("has contents, parsing")
		ir, err = s.parseIndexReport(req.GetContents())
	}
	if err != nil {
		return nil, err
	}
	zlog.Info(ctx).Msg("getting vulnerabilities")
	ccReport, err := s.matcher.GetVulnerabilities(ctx, ir)
	if err != nil {
		return nil, err
	}
	report, err := mappers.ToProtoV4VulnerabilityReport(ctx, ccReport)
	if err != nil {
		return nil, err
	}
	report.HashId = req.GetHashId()
	return report, nil
}

// retrieveIndexReport will pull an index report from the Indexer backend.
func (s *matcherService) retrieveIndexReport(ctx context.Context, hashID string) (*claircore.IndexReport, error) {
	ir, found, err := s.indexer.GetIndexReport(ctx, hashID)
	if err != nil {
		return nil, fmt.Errorf("internal error: %w", err)
	}
	if !found {
		return nil, errox.NotFound.CausedBy(err)
	}
	return ir, nil
}

// parseIndexReport will generate an index report from a Contents payload.
func (s *matcherService) parseIndexReport(contents *v4.Contents) (*claircore.IndexReport, error) {
	ir, err := mappers.ToClairCoreIndexReport(contents)
	if err != nil {
		// Validation should have captured all conversion errors.
		return nil, fmt.Errorf("internal error: %w", err)
	}
	return ir, nil
}

func (s *matcherService) GetMetadata(ctx context.Context, _ *types.Empty) (*v4.Metadata, error) {
	lastVulnUpdate, err := s.matcher.GetLastVulnerabilityUpdate(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting last vulnerability update time: %w", err)
	}

	timestamp, err := types.TimestampProto(lastVulnUpdate)
	if err != nil {
		return nil, fmt.Errorf("internal error: %w", err)
	}
	return &v4.Metadata{
		LastVulnerabilityUpdate: timestamp,
	}, nil
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *matcherService) RegisterServiceServer(grpcServer *grpc.Server) {
	v4.RegisterMatcherServer(grpcServer, s)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *matcherService) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	// TODO: Setup permissions for matcher.
	return ctx, allow.Anonymous().Authorized(ctx, fullMethodName)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *matcherService) RegisterServiceHandler(_ context.Context, _ *runtime.ServeMux, _ *grpc.ClientConn) error {
	// Currently we do not set up gRPC gateway for the matcher.
	return nil
}
