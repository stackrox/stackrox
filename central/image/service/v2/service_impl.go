package service

import (
	"context"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	imagev2DS "github.com/stackrox/rox/central/imagev2/datastore"
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
)

const (
	defaultPageSize = 100
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Image)): {
			v2.ImageExportService_ListImages_FullMethodName,
			v2.ImageExportService_ListScans_FullMethodName,
			v2.ImageExportService_ExportImages_FullMethodName,
			v2.ImageExportService_ExportScans_FullMethodName,
		},
	})
)

type serviceImpl struct {
	v2.UnimplementedImageExportServiceServer

	imageDS imagev2DS.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v2.RegisterImageExportServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v2.RegisterImageExportServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// ListImages returns a paginated list of image information (metadata, layers, components).
func (s *serviceImpl) ListImages(ctx context.Context, req *v2.ExportImagesRequest) (*v2.ListImagesResponse, error) {
	parsedQuery, err := search.ParseQuery(req.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(err, "parsing input query")
	}

	// Apply the since filter before pagination because ConjunctionQuery creates a new
	// top-level Query that does not carry the Pagination field from the inner query.
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

// ListScans returns a paginated list of image scan results and vulnerability findings.
// Each scan record is linked to its image via image_id.
func (s *serviceImpl) ListScans(ctx context.Context, req *v2.ExportScansRequest) (*v2.ListScansResponse, error) {
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

	results := make([]*v2.ImageScan, 0, len(images))
	for _, img := range images {
		results = append(results, toImageScan(img))
	}

	return &v2.ListScansResponse{
		Scans:      results,
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

	return s.imageDS.WalkByQuery(srv.Context(), filteredQuery, func(img *storage.ImageV2) error {
		return srv.Send(toImageInfo(img))
	})
}

// ExportScans streams image scan results and vulnerability findings for all images
// matching the query. Each streamed record is linked to its image via image_id.
func (s *serviceImpl) ExportScans(req *v2.ExportScansRequest, srv grpc.ServerStreamingServer[v2.ImageScan]) error {
	parsedQuery, err := search.ParseQuery(req.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return errors.Wrap(err, "parsing input query")
	}

	filteredQuery := applySinceFilter(parsedQuery, req.GetSince().AsTime())

	return s.imageDS.WalkByQuery(srv.Context(), filteredQuery, func(img *storage.ImageV2) error {
		return srv.Send(toImageScan(img))
	})
}

// applySinceFilter returns a query that adds a last_updated >= since predicate when since
// is non-zero. The new conjunction query is returned; the input query is not modified.
func applySinceFilter(q *v1.Query, since time.Time) *v1.Query {
	if since.IsZero() {
		return q
	}
	// Format the timestamp in the layout understood by the postgres time query parser.
	// The ">=" prefix generates the SQL predicate "lastupdated >= <since>".
	sinceStr := ">=" + since.UTC().Format("01/02/2006 3:04:05 PM MST")
	sinceQuery := search.NewQueryBuilder().AddStrings(search.LastUpdatedTime, sinceStr).ProtoQuery()
	return search.ConjunctionQuery(q, sinceQuery)
}

// toImageInfo converts a storage.ImageV2 to the v2 API ImageInfo projection.
// Components are included but their vulnerability lists are intentionally omitted; use
// toImageScan to retrieve vulnerability findings.
func toImageInfo(img *storage.ImageV2) *v2.ImageInfo {
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
		Digest:          img.GetDigest(),
		Names:           names,
		OperatingSystem: img.GetScan().GetOperatingSystem(),
		Created:         img.GetMetadata().GetV1().GetCreated(),
		Layers:          layers,
		Components:      components,
		LastUpdated:     img.GetLastUpdated(),
	}
}

// toImageScan converts a storage.ImageV2 to the v2 API ImageScan projection.
// All CVE findings are flattened from the per-component vulnerability lists.
// The ImageId field links the scan result back to the corresponding image.
func toImageScan(img *storage.ImageV2) *v2.ImageScan {
	var cves []*v2.ImageCVEFinding
	for _, comp := range img.GetScan().GetComponents() {
		for _, vuln := range comp.GetVulns() {
			cves = append(cves, &v2.ImageCVEFinding{
				Cve:                  vuln.GetCve(),
				Severity:             convertSeverity(vuln.GetSeverity()),
				Cvss:                 vuln.GetCvss(),
				IsFixable:            vuln.GetSetFixedBy() != nil,
				FixedBy:              vuln.GetFixedBy(),
				ComponentName:        comp.GetName(),
				ComponentVersion:     comp.GetVersion(),
				FirstImageOccurrence: vuln.GetFirstImageOccurrence(),
				State:                convertVulnState(vuln.GetState()),
				Summary:              vuln.GetSummary(),
				Link:                 vuln.GetLink(),
			})
		}
	}

	return &v2.ImageScan{
		ImageId:         img.GetId(),
		Digest:          img.GetDigest(),
		ScanTime:        img.GetScan().GetScanTime(),
		ScannerVersion:  img.GetScan().GetScannerVersion(),
		OperatingSystem: img.GetScan().GetOperatingSystem(),
		Cves:            cves,
		CveCounts:       toVulnCounts(img.GetScanStats()),
		LastUpdated:     img.GetLastUpdated(),
	}
}

// toVulnCounts converts the ScanStats cached counts to the API ImageVulnCounts message.
func toVulnCounts(stats *storage.ImageV2_ScanStats) *v2.ImageVulnCounts {
	if stats == nil {
		return &v2.ImageVulnCounts{}
	}
	return &v2.ImageVulnCounts{
		CriticalTotal:    stats.GetCriticalCveCount(),
		CriticalFixable:  stats.GetFixableCriticalCveCount(),
		ImportantTotal:   stats.GetImportantCveCount(),
		ImportantFixable: stats.GetFixableImportantCveCount(),
		ModerateTotal:    stats.GetModerateCveCount(),
		ModerateFixable:  stats.GetFixableModerateCveCount(),
		LowTotal:         stats.GetLowCveCount(),
		LowFixable:       stats.GetFixableLowCveCount(),
		UnknownTotal:     stats.GetUnknownCveCount(),
		UnknownFixable:   stats.GetFixableUnknownCveCount(),
	}
}

// convertSeverity maps storage.VulnerabilitySeverity to the v2 API enum.
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

// convertVulnState maps storage.VulnerabilityState to the v2 API enum.
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
