package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	complianceDS "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	"github.com/stackrox/rox/central/convert/storagetov2"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"google.golang.org/grpc"
)

const (
	maxPaginationLimit = 1000
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Compliance)): {
			"/v2.ComplianceResultsService/GetComplianceScanResultsOverview",
			"/v2.ComplianceResultsService/GetComplianceScanResults",
			"/v2.ComplianceResultsService/GetComplianceProfileScanStats",
			"/v2.ComplianceResultsService/GetComplianceClusterScanStats",
			"/v2.ComplianceResultsService/GetComplianceScanResultsCount",
			"/v2.ComplianceResultsService/GetComplianceOverallClusterStats",
			"/v2.ComplianceResultsService/GetComplianceOverallClusterCount",
			"/v2.ComplianceResultsService/GetComplianceScanCheckResult",
		},
	})
)

// New returns a service object for registering with grpc.
func New(complianceResultsDS complianceDS.DataStore) Service {
	return &serviceImpl{
		complianceResultsDS: complianceResultsDS,
	}
}

type serviceImpl struct {
	v2.UnimplementedComplianceResultsServiceServer

	complianceResultsDS complianceDS.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v2.RegisterComplianceResultsServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v2.RegisterComplianceResultsServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetComplianceScanResultsOverview lists current scan configurations with most recent results overview that match the query
func (s *serviceImpl) GetComplianceScanResultsOverview(_ context.Context, _ *v2.RawQuery) (*v2.ListComplianceScanResultsOverviewResponse, error) {
	return nil, errox.NotImplemented
}

// GetComplianceScanResults retrieves the most recent compliance operator scan results for the specified query
// TODO(ROX-20333):  the most recent portion will come when this ticket is worked once everything is wired up so we can tell
// what the latest scan is.
func (s *serviceImpl) GetComplianceScanResults(ctx context.Context, query *v2.RawQuery) (*v2.ListComplianceScanResultsResponse, error) {
	// Fill in Query.
	parsedQuery, err := search.ParseQuery(query.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to parse query %v", err)
	}

	// Fill in pagination.
	paginated.FillPaginationV2(parsedQuery, query.GetPagination(), maxPaginationLimit)

	scanResults, err := s.complianceResultsDS.SearchComplianceCheckResults(ctx, parsedQuery)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance scan results for query %v", query)
	}

	return &v2.ListComplianceScanResultsResponse{
		ScanResults: storagetov2.ComplianceV2CheckResults(scanResults),
	}, nil
}

// GetComplianceProfileScanStats lists current scan stats grouped by profile
func (s *serviceImpl) GetComplianceProfileScanStats(_ context.Context, _ *v2.RawQuery) (*v2.ListComplianceProfileScanStatsResponse, error) {
	// TODO(ROX-18102):  Need profiles stored first
	return nil, errox.NotImplemented
}

// GetComplianceClusterScanStats lists current scan stats grouped by cluster
func (s *serviceImpl) GetComplianceClusterScanStats(ctx context.Context, query *v2.RawQuery) (*v2.ListComplianceClusterScanStatsResponse, error) {
	// Fill in Query.
	parsedQuery, err := search.ParseQuery(query.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to parse query %v", err)
	}

	// Fill in pagination.
	paginated.FillPaginationV2(parsedQuery, query.GetPagination(), maxPaginationLimit)

	scanResults, err := s.complianceResultsDS.ComplianceCheckResultStats(ctx, parsedQuery)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance cluster scan stats for query %v", query)
	}

	return &v2.ListComplianceClusterScanStatsResponse{
		ScanStats: storagetov2.ComplianceV2ClusterStats(scanResults),
	}, nil
}

// GetComplianceOverallClusterStats lists current scan stats grouped by cluster
func (s *serviceImpl) GetComplianceOverallClusterStats(ctx context.Context, query *v2.RawQuery) (*v2.ListComplianceClusterOverallStatsResponse, error) {
	// Fill in Query.
	parsedQuery, err := search.ParseQuery(query.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to parse query %v", err)
	}

	// Fill in pagination.
	paginated.FillPaginationV2(parsedQuery, query.GetPagination(), maxPaginationLimit)

	scanResults, err := s.complianceResultsDS.ComplianceClusterStats(ctx, parsedQuery)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance cluster scan stats for query %v", query)
	}

	return &v2.ListComplianceClusterOverallStatsResponse{
		ScanStats: storagetov2.ComplianceV2ClusterOverallStats(scanResults),
	}, nil
}

// GetComplianceScanResultsCount returns scan results count
func (s *serviceImpl) GetComplianceScanResultsCount(ctx context.Context, query *v2.RawQuery) (*v2.CountComplianceScanResults, error) {
	parsedQuery, err := search.ParseQuery(query.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to parse query %v", err)
	}

	count, err := s.complianceResultsDS.CountCheckResults(ctx, parsedQuery)
	if err != nil {
		return nil, errors.Errorf("Unable to retrieve compliance scan results count for query %v", query)
	}
	return &v2.CountComplianceScanResults{
		Count: int32(count),
	}, nil
}

// GetComplianceOverallClusterCount returns scan results count
func (s *serviceImpl) GetComplianceOverallClusterCount(ctx context.Context, query *v2.RawQuery) (*v2.CountComplianceScanResults, error) {
	parsedQuery, err := search.ParseQuery(query.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to parse query %v", err)
	}

	count, err := s.complianceResultsDS.ComplianceClusterStatsCount(ctx, parsedQuery)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance scan results count for query %v", query)
	}
	return &v2.CountComplianceScanResults{
		Count: int32(count),
	}, nil
}

// GetComplianceScanCheckResult returns the specific result by ID
func (s *serviceImpl) GetComplianceScanCheckResult(ctx context.Context, req *v2.ResourceByID) (*v2.ComplianceCheckResult, error) {
	if req.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "compliance check result ID is required for retrieval")
	}

	scanResult, found, err := s.complianceResultsDS.GetComplianceCheckResult(ctx, req.GetId())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve compliance check result with id %q.", req.GetId())
	}
	if !found {
		return nil, errors.Wrapf(errox.NotFound, "compliance check result with id %q does not exist", req.GetId())
	}

	return storagetov2.ComplianceV2CheckResult(scanResult), nil
}
