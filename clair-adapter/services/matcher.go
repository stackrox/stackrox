package services

import (
	"context"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/clair-adapter/enricher/csaf"
	"github.com/stackrox/rox/clair-adapter/mappers"
	"github.com/stackrox/rox/clair-adapter/matcher"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// MatcherService implements the Scanner V4 MatcherServer interface.
type MatcherService struct {
	v4.UnimplementedMatcherServer
	m matcher.Matcher
}

// NewMatcherService creates a new MatcherService with the given matcher.
func NewMatcherService(m matcher.Matcher) *MatcherService {
	return &MatcherService{
		m: m,
	}
}

// GetVulnerabilities retrieves vulnerability information for a previously indexed manifest.
func (s *MatcherService) GetVulnerabilities(ctx context.Context, req *v4.GetVulnerabilitiesRequest) (*v4.VulnerabilityReport, error) {
	vr, enrichResult, err := s.m.GetVulnerabilities(ctx, req.GetHashId())
	if err != nil {
		// Check if it's a "not found" error from Clair client (HTTP 404)
		if strings.Contains(err.Error(), "HTTP 404") {
			return nil, status.Error(codes.NotFound, "vulnerability report not found")
		}
		return nil, status.Error(codes.Internal, errors.Wrap(err, "failed to get vulnerabilities").Error())
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
		return nil, status.Error(codes.Internal, errors.Wrap(err, "failed to convert vulnerability report").Error())
	}

	return protoReport, nil
}

// GetMetadata returns metadata about the vulnerability database, including the last update time.
func (s *MatcherService) GetMetadata(ctx context.Context, _ *emptypb.Empty) (*v4.Metadata, error) {
	ts, err := s.m.GetLastVulnerabilityUpdate(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, errors.Wrap(err, "failed to get metadata").Error())
	}

	return &v4.Metadata{
		LastVulnerabilityUpdate: timestamppb.New(ts),
	}, nil
}

// GetSBOM is not implemented.
func (s *MatcherService) GetSBOM(ctx context.Context, req *v4.GetSBOMRequest) (*v4.GetSBOMResponse, error) {
	return nil, status.Error(codes.Unimplemented, "GetSBOM is not implemented")
}

// ScanSBOM is not implemented.
func (s *MatcherService) ScanSBOM(ctx context.Context, req *v4.ScanSBOMRequest) (*v4.ScanSBOMResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ScanSBOM is not implemented")
}

// RegisterServiceServer registers the MatcherService with the gRPC server.
func (s *MatcherService) RegisterServiceServer(grpcServer *grpc.Server) {
	v4.RegisterMatcherServer(grpcServer, s)
}

// RegisterServiceHandler is a no-op for MatcherService (no HTTP gateway needed).
func (s *MatcherService) RegisterServiceHandler(_ context.Context, _ *runtime.ServeMux, _ *grpc.ClientConn) error {
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
