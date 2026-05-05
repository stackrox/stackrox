package service

import (
	"context"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/views/cveexport"
	"github.com/stackrox/rox/central/views/vulnfinding"
	v1 "github.com/stackrox/rox/generated/api/v1"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const defaultPageSize = 100

var authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
	user.With(permissions.View(resources.Image)): {
		v2.ImageExportService_ListImages_FullMethodName,
		v2.ImageExportService_ListFindings_FullMethodName,
		v2.ImageExportService_ListCVEs_FullMethodName,
		v2.ImageExportService_ExportImages_FullMethodName,
		v2.ImageExportService_ExportFindings_FullMethodName,
		v2.ImageExportService_ExportCVEs_FullMethodName,
	},
})

type serviceImpl struct {
	v2.UnimplementedImageExportServiceServer

	imageDS     datastore.DataStore
	cveView     cveexport.CveExportView
	findingView vulnfinding.FindingView
}

func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v2.RegisterImageExportServiceServer(grpcServer, s)
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v2.RegisterImageExportServiceHandler(ctx, mux, conn)
}

func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// ListImages returns a paginated list of image information (metadata, layers, components).
func (s *serviceImpl) ListImages(ctx context.Context, req *v2.ExportImagesRequest) (*v2.ListImagesResponse, error) {
	parsedQuery, err := search.ParseQuery(req.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(err, "parsing input query")
	}

	filteredQuery := applySinceFilter(parsedQuery, req.GetSince().AsTime())
	paginated.FillPaginationV2(filteredQuery, req.GetQuery().GetPagination(), defaultPageSize)

	countQuery := filteredQuery.CloneVT()
	countQuery.Pagination = nil
	totalCount, err := s.imageDS.Count(ctx, countQuery)
	if err != nil {
		return nil, errors.Wrap(err, "counting images")
	}

	images, err := s.imageDS.SearchRawImages(ctx, filteredQuery)
	if err != nil {
		return nil, errors.Wrap(err, "searching images")
	}

	results := make([]*v2.ImageInfo, 0, len(images))
	for _, img := range images {
		results = append(results, toImageInfo(img))
	}

	return &v2.ListImagesResponse{
		Images:     results,
		TotalCount: int32(totalCount),
	}, nil
}

// ListCVEs returns a paginated list of unique CVE details via SQL aggregation.
func (s *serviceImpl) ListCVEs(ctx context.Context, req *v2.ExportCVEsRequest) (*v2.ListCVEsResponse, error) {
	parsedQuery, err := search.ParseQuery(req.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(err, "parsing input query")
	}

	filteredQuery := applySinceCVEFilter(parsedQuery, req.GetSince().AsTime())
	paginated.FillPaginationV2(filteredQuery, req.GetQuery().GetPagination(), defaultPageSize)

	countQuery := filteredQuery.CloneVT()
	countQuery.Pagination = nil
	totalCount, err := s.cveView.Count(ctx, countQuery)
	if err != nil {
		return nil, errors.Wrap(err, "counting CVEs")
	}

	cves, err := s.cveView.Get(ctx, filteredQuery)
	if err != nil {
		return nil, errors.Wrap(err, "querying CVEs")
	}

	results := make([]*v2.CVEDetail, 0, len(cves))
	for _, c := range cves {
		results = append(results, cveExportToDetail(c))
	}

	return &v2.ListCVEsResponse{
		Cves:       results,
		TotalCount: int32(totalCount),
	}, nil
}

// ListFindings returns a paginated list of vulnerability findings via SQL JOIN.
func (s *serviceImpl) ListFindings(ctx context.Context, req *v2.ExportFindingsRequest) (*v2.ListFindingsResponse, error) {
	parsedQuery, err := search.ParseQuery(req.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(err, "parsing input query")
	}

	filteredQuery := applySinceFilter(parsedQuery, req.GetSince().AsTime())
	paginated.FillPaginationV2(filteredQuery, req.GetQuery().GetPagination(), defaultPageSize)

	countQuery := filteredQuery.CloneVT()
	countQuery.Pagination = nil
	totalCount, err := s.findingView.Count(ctx, countQuery)
	if err != nil {
		return nil, errors.Wrap(err, "counting findings")
	}

	findings, err := s.findingView.Get(ctx, filteredQuery)
	if err != nil {
		return nil, errors.Wrap(err, "querying findings")
	}

	results := make([]*v2.VulnerabilityFinding, 0, len(findings))
	for _, f := range findings {
		results = append(results, findingToProto(f))
	}

	return &v2.ListFindingsResponse{
		Findings:   results,
		TotalCount: int32(totalCount),
	}, nil
}

// ExportImages streams image information for all images matching the query.
func (s *serviceImpl) ExportImages(req *v2.ExportImagesRequest, srv grpc.ServerStreamingServer[v2.ImageInfo]) error {
	parsedQuery, err := search.ParseQuery(req.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return errors.Wrap(err, "parsing input query")
	}

	filteredQuery := applySinceFilter(parsedQuery, req.GetSince().AsTime())

	return s.imageDS.WalkByQuery(srv.Context(), filteredQuery, func(img *storage.Image) error {
		return srv.Send(toImageInfo(img))
	})
}

// ExportCVEs streams unique CVE details via SQL aggregation.
func (s *serviceImpl) ExportCVEs(req *v2.ExportCVEsRequest, srv grpc.ServerStreamingServer[v2.CVEDetail]) error {
	parsedQuery, err := search.ParseQuery(req.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return errors.Wrap(err, "parsing input query")
	}

	filteredQuery := applySinceCVEFilter(parsedQuery, req.GetSince().AsTime())

	cves, err := s.cveView.Get(srv.Context(), filteredQuery)
	if err != nil {
		return errors.Wrap(err, "querying CVEs")
	}
	for _, c := range cves {
		if sendErr := srv.Send(cveExportToDetail(c)); sendErr != nil {
			return sendErr
		}
	}
	return nil
}

// ExportFindings streams vulnerability findings via SQL JOIN.
func (s *serviceImpl) ExportFindings(req *v2.ExportFindingsRequest, srv grpc.ServerStreamingServer[v2.VulnerabilityFinding]) error {
	parsedQuery, err := search.ParseQuery(req.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return errors.Wrap(err, "parsing input query")
	}

	filteredQuery := applySinceFilter(parsedQuery, req.GetSince().AsTime())

	findings, err := s.findingView.Get(srv.Context(), filteredQuery)
	if err != nil {
		return errors.Wrap(err, "querying findings")
	}
	for _, f := range findings {
		if sendErr := srv.Send(findingToProto(f)); sendErr != nil {
			return sendErr
		}
	}
	return nil
}

// cveExportToDetail converts a SQL-aggregated CveExport to the API CVEDetail message.
func cveExportToDetail(c cveexport.CveExport) *v2.CVEDetail {
	detail := &v2.CVEDetail{
		Cve:             c.GetCVE(),
		Severity:        convertSeverity(c.GetSeverity()),
		Cvss:            c.GetCVSS(),
		Summary:         c.GetSummary(),
		Link:            c.GetLink(),
		EpssProbability: c.GetEPSSProbability(),
		EpssPercentile:  c.GetEPSSPercentile(),
	}
	if t := c.GetPublishedOn(); t != nil {
		detail.PublishedOn = timestamppb.New(*t)
	}
	if name := c.GetAdvisoryName(); name != "" {
		detail.Advisory = &v2.Advisory{
			Name: name,
			Link: c.GetAdvisoryLink(),
		}
	}
	return detail
}

// findingToProto converts a SQL-queried Finding to the API VulnerabilityFinding message.
func findingToProto(f vulnfinding.Finding) *v2.VulnerabilityFinding {
	return &v2.VulnerabilityFinding{
		DeploymentId:     f.GetDeploymentID(),
		ImageId:          f.GetImageID(),
		Cve:              f.GetCVE(),
		ComponentName:    f.GetComponentName(),
		ComponentVersion: f.GetComponentVersion(),
		IsFixable:        f.GetIsFixable(),
		FixedBy:          f.GetFixedBy(),
		State:            convertVulnState(f.GetState()),
		Severity:         convertSeverity(f.GetSeverity()),
		Cvss:             f.GetCVSS(),
		RepositoryCpe:    f.GetRepositoryCPE(),
	}
}

func applySinceFilter(q *v1.Query, since time.Time) *v1.Query {
	if since.IsZero() {
		return q
	}
	sinceStr := ">=" + since.UTC().Format("01/02/2006 3:04:05 PM MST")
	sinceQuery := search.NewQueryBuilder().AddStrings(search.LastUpdatedTime, sinceStr).ProtoQuery()
	return search.ConjunctionQuery(q, sinceQuery)
}

func applySinceCVEFilter(q *v1.Query, since time.Time) *v1.Query {
	if since.IsZero() {
		return q
	}
	sinceStr := ">=" + since.UTC().Format("01/02/2006 3:04:05 PM MST")
	sinceQuery := search.NewQueryBuilder().AddStrings(search.CVECreatedTime, sinceStr).ProtoQuery()
	return search.ConjunctionQuery(q, sinceQuery)
}

// toImageInfo converts a storage.Image to the v2 API ImageInfo projection.
func toImageInfo(img *storage.Image) *v2.ImageInfo {
	layers := make([]*v2.ImageExportLayer, 0, len(img.GetMetadata().GetV1().GetLayers()))
	for _, l := range img.GetMetadata().GetV1().GetLayers() {
		layers = append(layers, &v2.ImageExportLayer{
			Instruction: l.GetInstruction(),
			Value:       l.GetValue(),
		})
	}

	components := make([]*v2.ImageExportComponent, 0, len(img.GetScan().GetComponents()))
	for _, c := range img.GetScan().GetComponents() {
		components = append(components, &v2.ImageExportComponent{
			Name:     c.GetName(),
			Version:  c.GetVersion(),
			Location: c.GetLocation(),
			Source:   c.GetSource().String(),
		})
	}

	names := make([]string, 0, 1)
	if fullName := img.GetName().GetFullName(); fullName != "" {
		names = append(names, fullName)
	}

	return &v2.ImageInfo{
		Id:              img.GetId(),
		Digest:          img.GetId(),
		Names:           names,
		OperatingSystem: img.GetScan().GetOperatingSystem(),
		Created:         img.GetMetadata().GetV1().GetCreated(),
		Layers:          layers,
		Components:      components,
		LastUpdated:     img.GetLastUpdated(),
	}
}

func convertSeverity(s storage.VulnerabilitySeverity) v2.VulnerabilitySeverity {
	switch s {
	case storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY:
		return v2.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY
	case storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY:
		return v2.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY
	case storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY:
		return v2.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY
	case storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY:
		return v2.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY
	default:
		return v2.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
	}
}

func convertVulnState(s storage.VulnerabilityState) v2.VulnerabilityState {
	switch s {
	case storage.VulnerabilityState_DEFERRED:
		return v2.VulnerabilityState_DEFERRED
	case storage.VulnerabilityState_FALSE_POSITIVE:
		return v2.VulnerabilityState_FALSE_POSITIVE
	default:
		return v2.VulnerabilityState_OBSERVED
	}
}
