package services

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/quay/claircore"
	"github.com/quay/zlog"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/scanner/indexer"
	"github.com/stackrox/rox/scanner/mappers"
	"github.com/stackrox/rox/scanner/matcher"
	"github.com/stackrox/rox/scanner/services/validators"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc"
)

var matcherAuth = perrpc.FromMap(map[authz.Authorizer][]string{
	idcheck.CentralOnly(): {
		"/scanner.v4.Matcher/GetVulnerabilities",
		"/scanner.v4.Matcher/GetMetadata",
	},
})

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
	ctx = zlog.ContextWithValues(ctx, "component", "scanner/service/matcher.GetVulnerabilities")
	if err := validators.ValidateGetVulnerabilitiesRequest(req); err != nil {
		return nil, errox.InvalidArgs.CausedBy(err)
	}
	ctx = zlog.ContextWithValues(ctx, "hash_id", req.GetHashId())
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
	zlog.Info(ctx).Msgf("getting vulnerabilities for index report %q", req.GetHashId())
	ccReport, err := s.matcher.GetVulnerabilities(ctx, ir)
	if err != nil {
		zlog.Error(ctx).Err(err).Send()
		return nil, err
	}
	report, err := mappers.ToProtoV4VulnerabilityReport(ctx, ccReport)
	if err != nil {
		zlog.Error(ctx).Err(err).Msg("internal error: converting to v4.VulnerabilityReport")
		return nil, err
	}
	report.HashId = req.GetHashId()
	report.Notes = s.notes(ctx, report)
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
		// TODO(ROX-21362): Set scanner version.
		LastVulnerabilityUpdate: timestamp,
	}, nil
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *matcherService) RegisterServiceServer(grpcServer *grpc.Server) {
	v4.RegisterMatcherServer(grpcServer, s)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *matcherService) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	auth := matcherAuth
	// If this a dev build, allow anonymous traffic for testing purposes.
	if !buildinfo.ReleaseBuild {
		auth = allow.Anonymous()
	}
	return ctx, auth.Authorized(ctx, fullMethodName)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *matcherService) RegisterServiceHandler(_ context.Context, _ *runtime.ServeMux, _ *grpc.ClientConn) error {
	// Currently we do not set up gRPC gateway for the matcher.
	return nil
}

func (s *matcherService) notes(ctx context.Context, vr *v4.VulnerabilityReport) []v4.VulnerabilityReport_Note {
	if len(vr.Contents.Distributions) != 1 {
		return []v4.VulnerabilityReport_Note{v4.VulnerabilityReport_NOTE_OS_UNKNOWN}
	}

	dists := s.matcher.GetKnownDistributions(ctx)
	dist := vr.Contents.Distributions[0]
	dID := dist.GetDid()
	versionID := dist.GetVersionId()
	known := slices.ContainsFunc(dists, func(dist claircore.Distribution) bool {
		vID := mappers.VersionID(&dist)
		return dist.DID == dID && vID == versionID
	})
	if !known {
		return []v4.VulnerabilityReport_Note{v4.VulnerabilityReport_NOTE_OS_UNSUPPORTED}
	}

	return nil
}
