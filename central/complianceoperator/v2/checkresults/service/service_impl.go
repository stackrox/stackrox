package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	complianceDS "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	complianceIntegrationDS "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore"
	profileDatastore "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore"
	complianceRuleDS "github.com/stackrox/rox/central/complianceoperator/v2/rules/datastore"
	complianceConfigDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	"github.com/stackrox/rox/central/convert/storagetov2"
	v1 "github.com/stackrox/rox/generated/api/v1"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
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
			"/v2.ComplianceResultsService/GetComplianceScanResults",
			"/v2.ComplianceResultsService/GetComplianceProfileStats",
			"/v2.ComplianceResultsService/GetComplianceProfilesStats",
			"/v2.ComplianceResultsService/GetComplianceClusterScanStats",
			"/v2.ComplianceResultsService/GetComplianceScanResultsCount",
			"/v2.ComplianceResultsService/GetComplianceOverallClusterStats",
			"/v2.ComplianceResultsService/GetComplianceOverallClusterCount",
			"/v2.ComplianceResultsService/GetComplianceScanCheckResult",
			"/v2.ComplianceResultsService/GetComplianceScanConfigurationResults",
			"/v2.ComplianceResultsService/GetComplianceScanConfigurationResultsCount",
			"/v2.ComplianceResultsService/GetComplianceProfileResults",
			"/v2.ComplianceResultsService/GetComplianceClusterStats",
			"/v2.ComplianceResultsService/GetComplianceProfileCheckStats",
			"/v2.ComplianceResultsService/GetComplianceProfileCheckResult",
			"/v2.ComplianceResultsService/GetComplianceProfileClusterResults",
		},
	})
)

// New returns a service object for registering with grpc.
func New(complianceResultsDS complianceDS.DataStore, scanConfigDS complianceConfigDS.DataStore, integrationDS complianceIntegrationDS.DataStore, profileDS profileDatastore.DataStore, ruleDS complianceRuleDS.DataStore) Service {
	return &serviceImpl{
		complianceResultsDS: complianceResultsDS,
		scanConfigDS:        scanConfigDS,
		integrationDS:       integrationDS,
		profileDS:           profileDS,
		ruleDS:              ruleDS,
	}
}

type serviceImpl struct {
	v2.UnimplementedComplianceResultsServiceServer

	complianceResultsDS complianceDS.DataStore
	scanConfigDS        complianceConfigDS.DataStore
	integrationDS       complianceIntegrationDS.DataStore
	profileDS           profileDatastore.DataStore
	ruleDS              complianceRuleDS.DataStore
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

	return s.searchComplianceCheckResults(ctx, parsedQuery)
}

// GetComplianceProfileStats lists current scan stats grouped by the specified profile
func (s *serviceImpl) GetComplianceProfileStats(ctx context.Context, request *v2.ComplianceProfileResultsRequest) (*v2.ListComplianceProfileScanStatsResponse, error) {
	if request.GetProfileName() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Profile name is required")
	}

	// Fill in Query.
	parsedQuery, err := search.ParseQuery(request.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to parse query %v", err)
	}

	// Add the scan config name as an exact match
	parsedQuery = search.ConjunctionQuery(
		search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorProfileName, request.GetProfileName()).ProtoQuery(),
		parsedQuery,
	)

	// To get total count, need the parsed query without the paging.
	countQuery := parsedQuery.Clone()

	// Fill in pagination.
	paginated.FillPaginationV2(parsedQuery, request.GetQuery().GetPagination(), maxPaginationLimit)

	return s.getProfileStats(ctx, parsedQuery, countQuery)
}

// GetComplianceProfilesStats lists current scan stats grouped by profile
func (s *serviceImpl) GetComplianceProfilesStats(ctx context.Context, query *v2.RawQuery) (*v2.ListComplianceProfileScanStatsResponse, error) {
	// Fill in Query.
	parsedQuery, err := search.ParseQuery(query.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to parse query %v", err)
	}

	// To get total count, need the parsed query without the paging.
	countQuery := parsedQuery.Clone()

	// Fill in pagination.
	paginated.FillPaginationV2(parsedQuery, query.GetPagination(), maxPaginationLimit)

	return s.getProfileStats(ctx, parsedQuery, countQuery)
}

func (s *serviceImpl) getProfileStats(ctx context.Context, parsedQuery *v1.Query, countQuery *v1.Query) (*v2.ListComplianceProfileScanStatsResponse, error) {
	scanResults, err := s.complianceResultsDS.ComplianceProfileResultStats(ctx, parsedQuery)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to retrieve compliance profile scan stats for %+v", parsedQuery)
	}

	count, err := s.complianceResultsDS.CountByField(ctx, countQuery, search.ComplianceOperatorProfileName)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance scan results count for request %v", countQuery)
	}
	profileMap := map[string]*storage.ComplianceOperatorProfileV2{}
	for _, scan := range scanResults {
		profileResults, err := s.profileDS.SearchProfiles(ctx, search.NewQueryBuilder().
			AddExactMatches(search.ComplianceOperatorProfileName, scan.ProfileName).ProtoQuery())
		if err != nil {
			return nil, errors.Wrap(err, "Unable to retrieve compliance profile")
		}
		if len(profileResults) == 0 {
			return nil, errors.Errorf("Unable to retrieve compliance profile for %s", scan.ProfileName)
		}

		profileMap[scan.ProfileName] = profileResults[0]
	}

	return &v2.ListComplianceProfileScanStatsResponse{
		ScanStats:  storagetov2.ComplianceV2ProfileStats(scanResults, profileMap),
		TotalCount: int32(count),
	}, nil
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

	// Need to look up the scan config IDs to return with the results.
	scanConfigToIDs := make(map[string]string, len(scanResults))
	for _, result := range scanResults {
		if _, found := scanConfigToIDs[result.ScanConfigName]; !found {
			config, err := s.scanConfigDS.GetScanConfigurationByName(ctx, result.ScanConfigName)
			if err != nil {
				return nil, errors.Errorf("Unable to retrieve valid compliance scan configuration for results from %v", query)
			}
			scanConfigToIDs[result.ScanConfigName] = config.GetId()
		}
	}

	return &v2.ListComplianceClusterScanStatsResponse{
		ScanStats: storagetov2.ComplianceV2ClusterStats(scanResults, scanConfigToIDs),
	}, nil
}

// GetComplianceOverallClusterStats lists current scan stats grouped by cluster
func (s *serviceImpl) GetComplianceOverallClusterStats(ctx context.Context, query *v2.RawQuery) (*v2.ListComplianceClusterOverallStatsResponse, error) {
	// Fill in Query.
	parsedQuery, err := search.ParseQuery(query.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to parse query %v", err)
	}

	// To get total count, need the parsed query without the paging.
	countQuery := parsedQuery.Clone()

	// Fill in pagination.
	paginated.FillPaginationV2(parsedQuery, query.GetPagination(), maxPaginationLimit)

	scanResults, err := s.complianceResultsDS.ComplianceClusterStats(ctx, parsedQuery)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance cluster scan stats for query %v", query)
	}

	count, err := s.complianceResultsDS.CountByField(ctx, countQuery, search.ClusterID)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance scan results count for query %v", query)
	}

	// Lookup the integrations to get the status
	clusterErrors := make(map[string][]string, len(scanResults))
	for _, result := range scanResults {
		integrations, err := s.integrationDS.GetComplianceIntegrationByCluster(ctx, result.ClusterID)
		if err != nil || len(integrations) != 1 {
			return nil, errors.Errorf("Unable to retrieve cluster %q", result.ClusterID)
		}
		clusterErrors[result.ClusterID] = integrations[0].GetStatusErrors()
	}

	return &v2.ListComplianceClusterOverallStatsResponse{
		ScanStats:  storagetov2.ComplianceV2ClusterOverallStats(scanResults, clusterErrors),
		TotalCount: int32(count),
	}, nil
}

// GetComplianceClusterStats lists current scan stats grouped by cluster
func (s *serviceImpl) GetComplianceClusterStats(ctx context.Context, request *v2.ComplianceProfileResultsRequest) (*v2.ListComplianceClusterOverallStatsResponse, error) {
	if request.GetProfileName() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Profile name is required")
	}

	// Fill in Query.
	parsedQuery, err := search.ParseQuery(request.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to parse query %v", err)
	}

	if request.GetProfileName() != "" {
		// Add the profile name as an exact match
		parsedQuery = search.ConjunctionQuery(
			search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorProfileName, request.GetProfileName()).ProtoQuery(),
			parsedQuery,
		)
	}

	// To get total count, need the parsed query without the paging.
	countQuery := parsedQuery.Clone()

	// Fill in pagination.
	paginated.FillPaginationV2(parsedQuery, request.GetQuery().GetPagination(), maxPaginationLimit)

	scanResults, err := s.complianceResultsDS.ComplianceClusterStats(ctx, parsedQuery)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance cluster scan stats for request %v", request)
	}

	count, err := s.complianceResultsDS.CountByField(ctx, countQuery, search.ClusterID)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance scan results count for query %v", request)
	}

	// Lookup the integrations to get the status
	clusterErrors := make(map[string][]string, len(scanResults))
	for _, result := range scanResults {
		integrations, err := s.integrationDS.GetComplianceIntegrationByCluster(ctx, result.ClusterID)
		if err != nil || len(integrations) != 1 {
			return nil, errors.Errorf("Unable to retrieve cluster %q", result.ClusterID)
		}
		clusterErrors[result.ClusterID] = integrations[0].GetStatusErrors()
	}

	return &v2.ListComplianceClusterOverallStatsResponse{
		ScanStats:  storagetov2.ComplianceV2ClusterOverallStats(scanResults, clusterErrors),
		TotalCount: int32(count),
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

	count, err := s.complianceResultsDS.CountByField(ctx, parsedQuery, search.ClusterID)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance scan results count for query %v", query)
	}
	return &v2.CountComplianceScanResults{
		Count: int32(count),
	}, nil
}

// GetComplianceScanCheckResult returns the specific result by ID
func (s *serviceImpl) GetComplianceScanCheckResult(ctx context.Context, req *v2.ResourceByID) (*v2.ComplianceClusterCheckStatus, error) {
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

// GetComplianceScanConfigurationResults retrieves the most recent compliance operator scan results for the specified query
// TODO(ROX-20333):  the most recent portion will come when this ticket is worked once everything is wired up so we can tell
// what the latest scan is.
func (s *serviceImpl) GetComplianceScanConfigurationResults(ctx context.Context, request *v2.ComplianceScanResultsRequest) (*v2.ListComplianceScanResultsResponse, error) {
	if request.GetScanConfigName() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Scan configuration name is required")
	}

	// Fill in Query.
	parsedQuery, err := search.ParseQuery(request.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to parse query %v", err)
	}

	// Add the scan config name as an exact match
	parsedQuery = search.ConjunctionQuery(
		search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorScanConfigName, request.GetScanConfigName()).ProtoQuery(),
		parsedQuery,
	)

	// Fill in pagination.
	paginated.FillPaginationV2(parsedQuery, request.GetQuery().GetPagination(), maxPaginationLimit)

	return s.searchComplianceCheckResults(ctx, parsedQuery)
}

// GetComplianceScanConfigurationResultsCount returns scan results count
func (s *serviceImpl) GetComplianceScanConfigurationResultsCount(ctx context.Context, request *v2.ComplianceScanResultsRequest) (*v2.CountComplianceScanResults, error) {
	if request.GetScanConfigName() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Scan configuration name is required")
	}

	parsedQuery, err := search.ParseQuery(request.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to parse query %v", err)
	}

	// Add the scan config name as an exact match
	parsedQuery = search.ConjunctionQuery(
		search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorScanConfigName, request.GetScanConfigName()).ProtoQuery(),
		parsedQuery,
	)

	count, err := s.complianceResultsDS.CountCheckResults(ctx, parsedQuery)
	if err != nil {
		return nil, errors.Errorf("Unable to retrieve compliance scan results count for request %v", request)
	}
	return &v2.CountComplianceScanResults{
		Count: int32(count),
	}, nil
}

func (s *serviceImpl) GetComplianceProfileResults(ctx context.Context, request *v2.ComplianceProfileResultsRequest) (*v2.ListComplianceProfileResults, error) {
	if request.GetProfileName() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Profile name is required")
	}

	// Fill in Query.
	parsedQuery, err := search.ParseQuery(request.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to parse query %v", err)
	}

	// Add the profile name as an exact match
	parsedQuery = search.ConjunctionQuery(
		search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorProfileName, request.GetProfileName()).ProtoQuery(),
		parsedQuery,
	)

	// To get total count, need the parsed query without the paging.
	countQuery := parsedQuery.Clone()

	// Fill in pagination.
	paginated.FillPaginationV2(parsedQuery, request.GetQuery().GetPagination(), maxPaginationLimit)

	scanResults, err := s.complianceResultsDS.ComplianceProfileResults(ctx, parsedQuery)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to retrieve compliance profile scan stats for %+v", request)
	}

	count, err := s.complianceResultsDS.CountByField(ctx, countQuery, search.ComplianceOperatorCheckName)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance scan results count for query %v", request)
	}

	return &v2.ListComplianceProfileResults{
		ProfileResults: storagetov2.ComplianceV2ProfileResults(scanResults),
		ProfileName:    request.GetProfileName(),
		TotalCount:     int32(count),
	}, nil
}

// GetComplianceProfileCheckStats lists current scan stats grouped by the specified profile and compliance check
func (s *serviceImpl) GetComplianceProfileCheckStats(ctx context.Context, request *v2.ComplianceProfileCheckRequest) (*v2.ListComplianceProfileResults, error) {
	if request.GetProfileName() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Profile name is required")
	}

	if request.GetCheckName() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Compliance check name is required")
	}

	// Fill in Query.
	parsedQuery, err := search.ParseQuery(request.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to parse query %v", err)
	}

	// Add the scan config name as an exact match
	parsedQuery = search.ConjunctionQuery(
		search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorProfileName, request.GetProfileName()).
			AddExactMatches(search.ComplianceOperatorCheckName, request.GetCheckName()).
			ProtoQuery(),
		parsedQuery,
	)

	// Fill in pagination.
	paginated.FillPaginationV2(parsedQuery, request.GetQuery().GetPagination(), maxPaginationLimit)

	scanResults, err := s.complianceResultsDS.ComplianceProfileResults(ctx, parsedQuery)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to retrieve compliance profile check stats for %+v", request)
	}

	return &v2.ListComplianceProfileResults{
		ProfileResults: storagetov2.ComplianceV2ProfileResults(scanResults),
		ProfileName:    request.GetProfileName(),
		TotalCount:     int32(1),
	}, nil
}

// GetComplianceProfileCheckResult retrieves cluster status for a specific check result
func (s *serviceImpl) GetComplianceProfileCheckResult(ctx context.Context, request *v2.ComplianceProfileCheckRequest) (*v2.ListComplianceCheckClusterResponse, error) {
	if request.GetProfileName() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Profile name is required")
	}

	if request.GetCheckName() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Compliance check name is required")
	}

	// Fill in Query.
	parsedQuery, err := search.ParseQuery(request.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to parse query %v", err)
	}

	// Add the scan config name as an exact match
	parsedQuery = search.ConjunctionQuery(
		search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorProfileName, request.GetProfileName()).
			AddExactMatches(search.ComplianceOperatorCheckName, request.GetCheckName()).
			ProtoQuery(),
		parsedQuery,
	)

	// To get total count, need the parsed query without the paging.
	countQuery := parsedQuery.Clone()

	// Fill in pagination.
	paginated.FillPaginationV2(parsedQuery, request.GetQuery().GetPagination(), maxPaginationLimit)

	scanResults, err := s.complianceResultsDS.SearchComplianceCheckResults(ctx, parsedQuery)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance scan results for query %v", parsedQuery)
	}

	resultCount, err := s.complianceResultsDS.CountCheckResults(ctx, countQuery)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance scan results count for query %v", parsedQuery)
	}

	return &v2.ListComplianceCheckClusterResponse{
		CheckResults: storagetov2.ComplianceV2CheckClusterResults(scanResults),
		ProfileName:  request.GetProfileName(),
		CheckName:    request.GetCheckName(),
		TotalCount:   int32(resultCount),
	}, nil
}

// GetComplianceProfileClusterResults retrieves cluster status for a specific check result
func (s *serviceImpl) GetComplianceProfileClusterResults(ctx context.Context, request *v2.ComplianceProfileClusterRequest) (*v2.ListComplianceCheckResultResponse, error) {
	if request.GetProfileName() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Profile name is required")
	}

	if request.GetClusterId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Cluster ID is required")
	}

	// Fill in Query.
	parsedQuery, err := search.ParseQuery(request.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to parse query %v", err)
	}

	// Add the scan config name as an exact match
	parsedQuery = search.ConjunctionQuery(
		search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorProfileName, request.GetProfileName()).
			AddExactMatches(search.ClusterID, request.ClusterId).
			ProtoQuery(),
		parsedQuery,
	)

	countQuery := parsedQuery.Clone()

	// Fill in pagination.
	paginated.FillPaginationV2(parsedQuery, request.GetQuery().GetPagination(), maxPaginationLimit)

	scanResults, err := s.complianceResultsDS.SearchComplianceCheckResults(ctx, parsedQuery)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance scan results for query %v", parsedQuery)
	}

	checkToRule := make(map[string]string, len(scanResults))
	for _, result := range scanResults {
		rules, err := s.ruleDS.SearchRules(ctx, search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorRuleRef, result.GetRuleRefId()).ProtoQuery())
		if err != nil {
			return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance rule for query %v", parsedQuery)
		}
		if len(rules) != 1 {
			return nil, errors.Wrapf(errox.InvalidArgs, "Unable to process compliance rule for query %v", parsedQuery)
		}
		checkToRule[result.GetRuleRefId()] = rules[0].GetName()
	}

	resultCount, err := s.complianceResultsDS.CountCheckResults(ctx, countQuery)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance scan results count for query %v", parsedQuery)
	}

	return &v2.ListComplianceCheckResultResponse{
		CheckResults: storagetov2.ComplianceV2CheckResults(scanResults, checkToRule),
		ProfileName:  request.GetProfileName(),
		ClusterId:    request.GetClusterId(),
		TotalCount:   int32(resultCount),
	}, nil
}

func (s *serviceImpl) mapScanConfigToID(ctx context.Context, scanResults []*storage.ComplianceOperatorCheckResultV2) (map[string]string, error) {
	scanConfigToIDs := make(map[string]string, len(scanResults))
	for _, result := range scanResults {
		if _, found := scanConfigToIDs[result.ScanConfigName]; !found {
			config, err := s.scanConfigDS.GetScanConfigurationByName(ctx, result.ScanConfigName)
			if err != nil {
				return nil, errors.Errorf("Unable to retrieve valid compliance scan configuration %q", result.ScanConfigName)
			}
			scanConfigToIDs[result.ScanConfigName] = config.GetId()
		}
	}

	return scanConfigToIDs, nil
}

func (s *serviceImpl) searchComplianceCheckResults(ctx context.Context, parsedQuery *v1.Query) (*v2.ListComplianceScanResultsResponse, error) {
	scanResults, err := s.complianceResultsDS.SearchComplianceCheckResults(ctx, parsedQuery)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance scan results for query %v", parsedQuery)
	}

	// Need to look up the scan config IDs to return with the results.
	scanConfigToIDs, err := s.mapScanConfigToID(ctx, scanResults)
	if err != nil {
		return nil, err
	}

	return &v2.ListComplianceScanResultsResponse{
		ScanResults: storagetov2.ComplianceV2ScanResults(scanResults, scanConfigToIDs),
	}, nil
}
