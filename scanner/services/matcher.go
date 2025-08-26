package services

import (
	"context"
	"fmt"
	"slices"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/quay/claircore"
	"github.com/quay/zlog"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/scannerv4/mappers"
	"github.com/stackrox/rox/scanner/indexer"
	"github.com/stackrox/rox/scanner/matcher"
	"github.com/stackrox/rox/scanner/sbom"
	"github.com/stackrox/rox/scanner/services/validators"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var matcherAuth = perrpc.FromMap(map[authz.Authorizer][]string{
	idcheck.CentralOnly(): {
		v4.Matcher_GetVulnerabilities_FullMethodName,
		v4.Matcher_GetMetadata_FullMethodName,
		v4.Matcher_GetSBOM_FullMethodName,
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
	// anonymousAuthEnabled specifies if the service should allow for traffic from anonymous users.
	anonymousAuthEnabled bool
}

// NewMatcherService creates a new vulnerability matcher gRPC service, to enable
// empty content in enrich requests, pass a non-nil indexer.
func NewMatcherService(matcher matcher.Matcher, indexer indexer.ReportGetter) *matcherService {
	return &matcherService{
		matcher:              matcher,
		indexer:              indexer,
		disableEmptyContents: indexer == nil,
		anonymousAuthEnabled: env.ScannerV4AnonymousAuth.BooleanSetting(),
	}
}

func (s *matcherService) GetVulnerabilities(ctx context.Context, req *v4.GetVulnerabilitiesRequest) (*v4.VulnerabilityReport, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "scanner/service/matcher.GetVulnerabilities")
	if err := validators.ValidateGetVulnerabilitiesRequest(req); err != nil {
		return nil, errox.InvalidArgs.CausedBy(err)
	}
	if err := s.matcher.Initialized(ctx); err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "the matcher is not initialized: %v", err)
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
		ir, err = getClairIndexReport(ctx, s.indexer, req.GetHashId())
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

// parseIndexReport will generate an index report from a Contents payload.
func (s *matcherService) parseIndexReport(contents *v4.Contents) (*claircore.IndexReport, error) {
	ir, err := mappers.ToClairCoreIndexReport(contents)
	if err != nil {
		// Validation should have captured all conversion errors.
		return nil, fmt.Errorf("internal error: %w", err)
	}
	return ir, nil
}

func (s *matcherService) GetMetadata(ctx context.Context, _ *protocompat.Empty) (*v4.Metadata, error) {
	lastVulnUpdate, err := s.matcher.GetLastVulnerabilityUpdate(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting last vulnerability update time: %w", err)
	}

	timestamp, err := protocompat.ConvertTimeToTimestampOrError(lastVulnUpdate)
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
	if s.anonymousAuthEnabled {
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
	dists := vr.GetContents().GetDistributions()
	if len(dists) != 1 {
		return []v4.VulnerabilityReport_Note{v4.VulnerabilityReport_NOTE_OS_UNKNOWN}
	}

	dist := dists[0]
	distID := dist.GetDid()
	versionID := dist.GetVersionId()
	knownDists := s.matcher.GetKnownDistributions(ctx)
	known := slices.ContainsFunc(knownDists, func(dist claircore.Distribution) bool {
		vID := mappers.VersionID(&dist)
		return distID == dist.DID && versionID == vID
	})
	if !known {
		return []v4.VulnerabilityReport_Note{v4.VulnerabilityReport_NOTE_OS_UNSUPPORTED}
	}

	return nil
}

func (s *matcherService) GetSBOM(ctx context.Context, req *v4.GetSBOMRequest) (*v4.GetSBOMResponse, error) {
	ctx = zlog.ContextWithValues(ctx,
		"component", "scanner/service/matcher.GetSBOM",
		"id", req.GetId(),
		"name", req.GetName(),
	)

	if err := validators.ValidateGetSBOMRequest(req); err != nil {
		return nil, errox.InvalidArgs.CausedBy(err)
	}

	zlog.Info(ctx).Msgf("generating SBOM from index report (%d dists, %d envs, %d pkgs, %d repos)",
		len(req.GetContents().GetDistributions()),
		len(req.GetContents().GetEnvironments()),
		len(req.GetContents().GetPackages()),
		len(req.GetContents().GetRepositories()),
	)

	// The remote indexer is not used. This creates flexibility and enables SBOMs to be generated
	// from index reports not stored in the local indexer (such as from node scans and from things not
	// indexed by indexer, such as Central scans from third party scanners).
	ir, err := s.parseIndexReport(req.GetContents())
	if err != nil {
		zlog.Error(ctx).Err(err).Msg("parsing index report")
		return nil, err
	}

	sbom, err := s.matcher.GetSBOM(ctx, ir, &sbom.Options{
		Name:      req.GetId(),
		Namespace: req.GetUri(),
		Comment:   fmt.Sprintf("Tech Preview - generated for '%s'", req.GetName()),
	})
	if err != nil {
		zlog.Error(ctx).Err(err).Msg("generating SBOM")
		return nil, err
	}

	return &v4.GetSBOMResponse{Sbom: sbom}, nil
}
