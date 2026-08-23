package services

import (
	"context"
	"errors"
	"log/slog"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stackrox/rox/clair-adapter/clairclient"
	"github.com/stackrox/rox/clair-adapter/enricher/csaf"
	"github.com/stackrox/rox/clair-adapter/mappers"
	"github.com/stackrox/rox/clair-adapter/matcher"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var matcherAuth = perrpc.FromMap(map[authz.Authorizer][]string{
	idcheck.CentralOnly(): {
		v4.Matcher_GetVulnerabilities_FullMethodName,
		v4.Matcher_GetMetadata_FullMethodName,
		v4.Matcher_GetSBOM_FullMethodName,
		v4.Matcher_ScanSBOM_FullMethodName,
	},
})

// matcherService implements the Scanner V4 MatcherServer interface.
type matcherService struct {
	v4.UnimplementedMatcherServer
	m matcher.Matcher
}

// NewMatcherService creates a new matcherService with the given matcher.
func NewMatcherService(m matcher.Matcher) *matcherService {
	return &matcherService{
		m: m,
	}
}

// GetVulnerabilities retrieves vulnerability information for a previously indexed manifest.
func (s *matcherService) GetVulnerabilities(ctx context.Context, req *v4.GetVulnerabilitiesRequest) (*v4.VulnerabilityReport, error) {
	slog.InfoContext(ctx, "GetVulnerabilities called", "hash_id", req.GetHashId())

	vr, enrichResult, err := s.m.GetVulnerabilities(ctx, req.GetHashId())
	if err != nil {
		// Check if it's a "not found" error from Clair client
		if errors.Is(err, clairclient.ErrNotFound) {
			slog.InfoContext(ctx, "Vulnerability report not found", "hash_id", req.GetHashId())
			return nil, status.Error(codes.NotFound, "vulnerability report not found")
		}
		slog.ErrorContext(ctx, "Failed to get vulnerabilities", "error", err, "hash_id", req.GetHashId())
		return nil, status.Errorf(codes.Internal, "failed to get vulnerabilities: %v", err)
	}

	// Convert CSAF advisories from enricher format to mapper format
	mappersCSAF := convertCSAFAdvisories(enrichResult.CSAFAdvisories)

	// Convert to proto format with enrichments
	protoReport, err := mappers.ToProtoVulnerabilityReportWithEnrichments(
		ctx,
		vr,
		enrichResult.NVDVulns,
		enrichResult.EPSSItems,
		mappersCSAF,
		enrichResult.PkgFixedBy,
	)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to convert vulnerability report", "error", err, "hash_id", req.GetHashId())
		return nil, status.Errorf(codes.Internal, "failed to convert vulnerability report: %v", err)
	}

	slog.InfoContext(ctx, "GetVulnerabilities completed", "hash_id", req.GetHashId())
	return protoReport, nil
}

// GetMetadata returns metadata about the vulnerability database, including the last update time.
func (s *matcherService) GetMetadata(ctx context.Context, _ *emptypb.Empty) (*v4.Metadata, error) {
	slog.InfoContext(ctx, "GetMetadata called")

	ts, err := s.m.GetLastVulnerabilityUpdate(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get metadata", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to get metadata: %v", err)
	}

	slog.InfoContext(ctx, "GetMetadata completed", "last_vulnerability_update", ts)
	return &v4.Metadata{
		LastVulnerabilityUpdate: timestamppb.New(ts),
	}, nil
}

// GetSBOM is not implemented.
func (s *matcherService) GetSBOM(ctx context.Context, req *v4.GetSBOMRequest) (*v4.GetSBOMResponse, error) {
	slog.InfoContext(ctx, "GetSBOM called (not implemented)")
	return nil, status.Error(codes.Unimplemented, "GetSBOM is not implemented")
}

// ScanSBOM is not implemented.
func (s *matcherService) ScanSBOM(ctx context.Context, req *v4.ScanSBOMRequest) (*v4.ScanSBOMResponse, error) {
	slog.InfoContext(ctx, "ScanSBOM called (not implemented)")
	return nil, status.Error(codes.Unimplemented, "ScanSBOM is not implemented")
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *matcherService) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, matcherAuth.Authorized(ctx, fullMethodName)
}

// RegisterServiceServer registers the matcherService with the gRPC server.
func (s *matcherService) RegisterServiceServer(grpcServer *grpc.Server) {
	v4.RegisterMatcherServer(grpcServer, s)
}

// RegisterServiceHandler is a no-op for matcherService (no HTTP gateway needed).
func (s *matcherService) RegisterServiceHandler(_ context.Context, _ *runtime.ServeMux, _ *grpc.ClientConn) error {
	return nil
}

// convertCSAFAdvisories converts from enricher/csaf.Advisory to mappers.CSAFAdvisory.
func convertCSAFAdvisories(csafAdvisories map[string]*csaf.Advisory) map[string]*mappers.CSAFAdvisory {
	if csafAdvisories == nil {
		return nil
	}

	result := make(map[string]*mappers.CSAFAdvisory, len(csafAdvisories))
	for cveID, advisory := range csafAdvisories {
		result[cveID] = &mappers.CSAFAdvisory{
			Name:        advisory.Name,
			Description: advisory.Description,
			ReleaseDate: advisory.ReleaseDate,
			Severity:    advisory.Severity,
			CVSSv3: mappers.CVSSScore{
				BaseScore: advisory.CVSSv3.BaseScore,
				Vector:    advisory.CVSSv3.Vector,
			},
			CVSSv2: mappers.CVSSScore{
				BaseScore: advisory.CVSSv2.BaseScore,
				Vector:    advisory.CVSSv2.Vector,
			},
		}
	}

	return result
}
