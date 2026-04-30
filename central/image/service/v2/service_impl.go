package service

import (
	"context"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	imagecvev2DS "github.com/stackrox/rox/central/cve/image/v2/datastore"
	imageDS "github.com/stackrox/rox/central/image/datastore"
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
			v2.ImageExportService_ListCVEs_FullMethodName,
			v2.ImageExportService_ExportImages_FullMethodName,
			v2.ImageExportService_ExportScans_FullMethodName,
			v2.ImageExportService_ExportCVEs_FullMethodName,
		},
	})
)

type serviceImpl struct {
	v2.UnimplementedImageExportServiceServer

	// imageDS is the mapping datastore used by the v1 image service. It handles
	// both the legacy images table and the imagev2 table transparently, so it works
	// regardless of the ROX_FLATTEN_IMAGE_DATA feature flag.
	imageDS imageDS.DataStore
	cveDS   imagecvev2DS.DataStore
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

// ListCVEs returns a paginated list of unique CVE details. Uniqueness is by CVE
// identifier string. The implementation collects all matching CVEs, deduplicates
// in memory, then applies offset-based pagination on the deduplicated set.
func (s *serviceImpl) ListCVEs(ctx context.Context, req *v2.ExportCVEsRequest) (*v2.ListCVEsResponse, error) {
	parsedQuery, err := search.ParseQuery(req.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(err, "parsing input query")
	}

	filteredQuery := applySinceCVEFilter(parsedQuery, req.GetSince().AsTime())
	filteredQuery = sortByCVE(filteredQuery)

	// Collect all unique CVEs in a single pass.
	unique, err := collectUniqueCVEs(ctx, s.cveDS, filteredQuery)
	if err != nil {
		return nil, errors.Wrap(err, "collecting unique CVEs")
	}

	// Apply manual pagination on the deduplicated result.
	p := req.GetQuery().GetPagination()
	offset := int(p.GetOffset())
	limit := int(p.GetLimit())
	if limit <= 0 || limit > defaultPageSize {
		limit = defaultPageSize
	}
	start := min(offset, len(unique))
	end := min(start+limit, len(unique))

	return &v2.ListCVEsResponse{
		Cves:       unique[start:end],
		TotalCount: int32(len(unique)),
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

// ExportScans streams image scan results and vulnerability findings for all images
// matching the query. Each streamed record is linked to its image via image_id.
func (s *serviceImpl) ExportScans(req *v2.ExportScansRequest, srv grpc.ServerStreamingServer[v2.ImageScan]) error {
	parsedQuery, err := search.ParseQuery(req.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return errors.Wrap(err, "parsing input query")
	}

	filteredQuery := applySinceFilter(parsedQuery, req.GetSince().AsTime())

	return s.imageDS.WalkByQuery(srv.Context(), filteredQuery, func(img *storage.Image) error {
		return srv.Send(toImageScan(img))
	})
}

// ExportCVEs streams unique CVE details for all CVEs matching the query. Results
// are sorted by CVE identifier and deduplicated: each CVE is emitted exactly once.
func (s *serviceImpl) ExportCVEs(req *v2.ExportCVEsRequest, srv grpc.ServerStreamingServer[v2.CVEDetail]) error {
	parsedQuery, err := search.ParseQuery(req.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return errors.Wrap(err, "parsing input query")
	}

	filteredQuery := applySinceCVEFilter(parsedQuery, req.GetSince().AsTime())
	filteredQuery = sortByCVE(filteredQuery)

	var lastSeen string
	return s.cveDS.WalkByQuery(srv.Context(), filteredQuery, func(cve *storage.ImageCVEV2) error {
		if id := cve.GetCveBaseInfo().GetCve(); id != lastSeen {
			lastSeen = id
			return srv.Send(toCVEDetail(cve))
		}
		return nil
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

// applySinceCVEFilter filters CVEs by the time they were first seen in the system
// (CVEInfo.created_at). The "since" semantics differ from the image last_updated filter.
func applySinceCVEFilter(q *v1.Query, since time.Time) *v1.Query {
	if since.IsZero() {
		return q
	}
	sinceStr := ">=" + since.UTC().Format("01/02/2006 3:04:05 PM MST")
	sinceQuery := search.NewQueryBuilder().AddStrings(search.CVECreatedTime, sinceStr).ProtoQuery()
	return search.ConjunctionQuery(q, sinceQuery)
}

// sortByCVE adds an ascending sort on the CVE identifier string to a query.
// This enables the streaming deduplication in ExportCVEs to work with O(1) state.
func sortByCVE(q *v1.Query) *v1.Query {
	cloned := q.CloneVT()
	if cloned.Pagination == nil {
		cloned.Pagination = &v1.QueryPagination{}
	}
	cloned.Pagination.SortOptions = []*v1.QuerySortOption{
		{Field: search.CVE.String(), Reversed: false},
	}
	return cloned
}

// collectUniqueCVEs walks all CVEs matching q (must be sorted by CVE string) and
// returns one CVEDetail per unique CVE identifier.
func collectUniqueCVEs(ctx context.Context, ds imagecvev2DS.DataStore, q *v1.Query) ([]*v2.CVEDetail, error) {
	var (
		results  []*v2.CVEDetail
		lastSeen string
	)
	err := ds.WalkByQuery(ctx, q, func(cve *storage.ImageCVEV2) error {
		if id := cve.GetCveBaseInfo().GetCve(); id != lastSeen {
			lastSeen = id
			results = append(results, toCVEDetail(cve))
		}
		return nil
	})
	return results, err
}

// toCVEDetail converts a storage.ImageCVEV2 to the API CVEDetail message.
func toCVEDetail(cve *storage.ImageCVEV2) *v2.CVEDetail {
	return &v2.CVEDetail{
		Cve:             cve.GetCveBaseInfo().GetCve(),
		Severity:        convertSeverity(cve.GetSeverity()),
		Cvss:            cve.GetCvss(),
		Summary:         cve.GetCveBaseInfo().GetSummary(),
		Link:            cve.GetCveBaseInfo().GetLink(),
		PublishedOn:     cve.GetCveBaseInfo().GetPublishedOn(),
		EpssProbability: cve.GetCveBaseInfo().GetEpss().GetEpssProbability(),
	}
}

// toImageInfo converts a storage.Image to the v2 API ImageInfo projection.
// Components are included but their vulnerability lists are intentionally omitted; use
// toImageScan to retrieve vulnerability findings.
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
		// In the legacy image model the id field holds the image SHA, which is
		// also the digest. There is no separate UUID-style id.
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

// toImageScan converts a storage.Image to the v2 API ImageScan projection.
// CVE findings contain only the finding-specific fields (component reference, fixability,
// state). Full CVE metadata is available separately via the /cves endpoint.
func toImageScan(img *storage.Image) *v2.ImageScan {
	var cves []*v2.ImageCVEFinding
	for _, comp := range img.GetScan().GetComponents() {
		for _, vuln := range comp.GetVulns() {
			cves = append(cves, &v2.ImageCVEFinding{
				Cve:                  vuln.GetCve(),
				ComponentName:        comp.GetName(),
				ComponentVersion:     comp.GetVersion(),
				IsFixable:            vuln.GetSetFixedBy() != nil,
				FixedBy:              vuln.GetFixedBy(),
				FirstImageOccurrence: vuln.GetFirstImageOccurrence(),
				State:                convertVulnState(vuln.GetState()),
			})
		}
	}

	return &v2.ImageScan{
		// In the legacy model, id is the image SHA which also serves as the digest.
		ImageId:         img.GetId(),
		Digest:          img.GetId(),
		ScanTime:        img.GetScan().GetScanTime(),
		ScannerVersion:  img.GetScan().GetScannerVersion(),
		OperatingSystem: img.GetScan().GetOperatingSystem(),
		Cves:            cves,
		CveCounts:       computeVulnCounts(img.GetScan().GetComponents()),
		LastUpdated:     img.GetLastUpdated(),
	}
}

// computeVulnCounts tallies CVE counts per severity by iterating the scan components.
// storage.Image does not cache per-severity counts the way ImageV2.ScanStats does,
// so we compute them on the fly from the embedded vulnerability list.
func computeVulnCounts(components []*storage.EmbeddedImageScanComponent) *v2.ImageVulnCounts {
	counts := &v2.ImageVulnCounts{}
	for _, comp := range components {
		for _, vuln := range comp.GetVulns() {
			fixable := vuln.GetSetFixedBy() != nil
			switch vuln.GetSeverity() {
			case storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY:
				counts.CriticalTotal++
				if fixable {
					counts.CriticalFixable++
				}
			case storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY:
				counts.ImportantTotal++
				if fixable {
					counts.ImportantFixable++
				}
			case storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY:
				counts.ModerateTotal++
				if fixable {
					counts.ModerateFixable++
				}
			case storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY:
				counts.LowTotal++
				if fixable {
					counts.LowFixable++
				}
			default:
				counts.UnknownTotal++
				if fixable {
					counts.UnknownFixable++
				}
			}
		}
	}
	return counts
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
