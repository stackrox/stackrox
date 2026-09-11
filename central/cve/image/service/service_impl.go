package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	imageCVEV2DS "github.com/stackrox/rox/central/cve/image/v2/datastore"
	imageComponentV2DS "github.com/stackrox/rox/central/imagecomponent/v2/datastore"
	"github.com/stackrox/rox/central/views"
	"github.com/stackrox/rox/central/views/common"
	"github.com/stackrox/rox/central/views/imagecve"
	vulnReqDataStore "github.com/stackrox/rox/central/vulnmgmt/vulnerabilityrequest/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/set"
	"google.golang.org/grpc"
)

const maxImageCVEsReturned = 1000

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Image)): {
			v1.ImageCVEService_ListImageCVEs_FullMethodName,
			v1.ImageCVEService_CountImageCVEs_FullMethodName,
		},
	})
)

type serviceImpl struct {
	v1.UnimplementedImageCVEServiceServer

	cveView      imagecve.CveView
	imageCVEs    imageCVEV2DS.DataStore
	components   imageComponentV2DS.DataStore
	vulnReqStore vulnReqDataStore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterImageCVEServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterImageCVEServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// CountImageCVEs returns the number of distinct image CVEs matching the query.
// Wraps ImageCVEView.Count — the same call as GraphQL ImageCVECount.
func (s *serviceImpl) CountImageCVEs(ctx context.Context, request *v1.RawQuery) (*v1.CountImageCVEsResponse, error) {
	parsedQuery, err := search.ParseQuery(request.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}
	count, err := s.cveView.Count(ctx, parsedQuery)
	if err != nil {
		return nil, err
	}
	return &v1.CountImageCVEsResponse{Count: int32(count)}, nil
}

// ListImageCVEs returns aggregated image CVEs for the Workload CVE overview table.
// Wraps GraphQL imageCVEs (ImageCVEView.Get), distroTuples (ImageCVEV2 + component OS),
// and exceptionCount (unexpired vuln requests).
func (s *serviceImpl) ListImageCVEs(ctx context.Context, request *v1.ListImageCVEsRequest) (*v1.ListImageCVEsResponse, error) {
	parsedQuery, err := search.ParseQuery(request.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}
	paginated.FillPagination(parsedQuery, request.GetPagination(), maxImageCVEsReturned)

	cores, err := s.cveView.Get(ctx, parsedQuery, views.ReadOptions{})
	if err != nil {
		return nil, err
	}

	distrosByCVE, err := s.distroTuplesByCVE(ctx, cores)
	if err != nil {
		return nil, err
	}
	exceptionCounts, err := s.exceptionCounts(ctx, cores, request.GetRequestStatuses())
	if err != nil {
		return nil, err
	}

	out := make([]*v1.ImageCVE, 0, len(cores))
	for _, core := range cores {
		cve := core.GetCVE()
		out = append(out, &v1.ImageCVE{
			Cve:                          cve,
			TopCvss:                      core.GetTopCVSS(),
			TopNvdCvss:                   core.GetTopNVDCVSS(),
			AffectedImageCount:           int32(core.GetAffectedImageCount()),
			AffectedImageCountBySeverity: severityCounts(core.GetImagesBySeverity()),
			FirstDiscoveredInSystem:      protocompat.ConvertTimeToTimestampOrNil(core.GetFirstDiscoveredInSystem()),
			PublishedOn:                  protocompat.ConvertTimeToTimestampOrNil(core.GetPublishDate()),
			DistroTuples:                 distrosByCVE[cve],
			ExceptionCount:               exceptionCounts[cve],
		})
	}
	return &v1.ListImageCVEsResponse{ImageCves: out}, nil
}

func (s *serviceImpl) distroTuplesByCVE(ctx context.Context, cores []imagecve.CveCore) (map[string][]*v1.ImageCVEDistroTuple, error) {
	cveIDs := make([]string, 0)
	for _, core := range cores {
		cveIDs = append(cveIDs, core.GetCVEIDs()...)
	}
	if len(cveIDs) == 0 {
		return nil, nil
	}

	// Same query as GraphQL ImageCVECore.DistroTuples (image_cve_core.go).
	query := search.NewQueryBuilder().AddExactMatches(search.CVEID, cveIDs...).ProtoQuery()
	vulns, err := s.imageCVEs.SearchRawImageCVEs(ctx, query)
	if err != nil {
		return nil, err
	}

	osByComponent, err := s.operatingSystemsByComponent(ctx, vulns)
	if err != nil {
		return nil, err
	}

	out := make(map[string][]*v1.ImageCVEDistroTuple, len(cores))
	for _, vuln := range vulns {
		cveName := vuln.GetCveBaseInfo().GetCve()
		out[cveName] = append(out[cveName], &v1.ImageCVEDistroTuple{
			Summary:                    vuln.GetCveBaseInfo().GetSummary(),
			OperatingSystem:            osByComponent[vuln.GetComponentId()],
			Cvss:                       vuln.GetCvss(),
			ScoreVersion:               vuln.GetCveBaseInfo().GetScoreVersion().String(),
			NvdCvss:                    vuln.GetNvdcvss(),
			NvdScoreVersion:            vuln.GetNvdScoreVersion().String(),
			EpssProbability:            vuln.GetCveBaseInfo().GetEpss().GetEpssProbability(),
			KnownRansomwareCampaignUse: vuln.GetCveBaseInfo().GetExploit().GetKnownRansomwareCampaignUse(),
			Severity:                   vuln.GetSeverity(),
		})
	}
	return out, nil
}

func (s *serviceImpl) operatingSystemsByComponent(ctx context.Context, vulns []*storage.ImageCVEV2) (map[string]string, error) {
	componentIDs := set.NewStringSet()
	for _, vuln := range vulns {
		if id := vuln.GetComponentId(); id != "" {
			componentIDs.Add(id)
		}
	}
	if componentIDs.Cardinality() == 0 {
		return nil, nil
	}

	// GraphQL DistroTuples look up OS via ImageComponentV2 (image_vulnerabilities.go OperatingSystem).
	components, err := s.components.GetBatch(ctx, componentIDs.AsSlice())
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(components))
	for _, component := range components {
		out[component.GetId()] = component.GetOperatingSystem()
	}
	return out, nil
}

func (s *serviceImpl) exceptionCounts(ctx context.Context, cores []imagecve.CveCore, requestStatuses []string) (map[string]int32, error) {
	counts := make(map[string]int32, len(cores))
	for _, core := range cores {
		cve := core.GetCVE()
		if cve == "" {
			continue
		}
		// Same unexpired + CVE (+ optional request status) query as GraphQL ExceptionCount.
		qb := search.NewQueryBuilder().
			AddBools(search.ExpiredRequest, false).
			AddExactMatches(search.CVE, cve)
		if len(requestStatuses) > 0 {
			qb.AddExactMatches(search.RequestStatus, requestStatuses...)
		}
		count, err := s.vulnReqStore.Count(ctx, qb.ProtoQuery())
		if err != nil {
			if errors.Is(err, errox.NotAuthorized) {
				return counts, nil
			}
			return nil, err
		}
		counts[cve] = int32(count)
	}
	return counts, nil
}

func severityCounts(sev common.ResourceCountByCVESeverity) *v1.ResourceCountByCVESeverity {
	if sev == nil {
		return &v1.ResourceCountByCVESeverity{}
	}
	return &v1.ResourceCountByCVESeverity{
		Critical:  totalCount(sev.GetCriticalSeverityCount()),
		Important: totalCount(sev.GetImportantSeverityCount()),
		Moderate:  totalCount(sev.GetModerateSeverityCount()),
		Low:       totalCount(sev.GetLowSeverityCount()),
		Unknown:   totalCount(sev.GetUnknownSeverityCount()),
	}
}

func totalCount(c common.ResourceCountByFixability) int32 {
	if c == nil {
		return 0
	}
	return int32(c.GetTotal())
}
