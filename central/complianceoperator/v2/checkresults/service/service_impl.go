package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	complianceDS "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	"github.com/stackrox/rox/central/complianceoperator/v2/checkresults/utils"
	complianceIntegrationDS "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore"
	profileDatastore "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore"
	complianceRuleDS "github.com/stackrox/rox/central/complianceoperator/v2/rules/datastore"
	complianceConfigDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	complianceScanDS "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	"github.com/stackrox/rox/central/convert/storagetov2"
	v1 "github.com/stackrox/rox/generated/api/v1"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	types "github.com/stackrox/rox/pkg/protocompat"
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
			"/v2.ComplianceResultsService/GetComplianceOverallClusterCount",
			"/v2.ComplianceResultsService/GetComplianceScanCheckResult",
			"/v2.ComplianceResultsService/GetComplianceScanConfigurationResults",
			"/v2.ComplianceResultsService/GetComplianceScanConfigurationResultsCount",
			"/v2.ComplianceResultsService/GetComplianceProfileResults",
			"/v2.ComplianceResultsService/GetComplianceProfileCheckResult",
			"/v2.ComplianceResultsService/GetComplianceProfileClusterResults",
			"/v2.ComplianceResultsService/GetComplianceProfileCheckDetails",
		},
	})
)

// New returns a service object for registering with grpc.
func New(complianceResultsDS complianceDS.DataStore, scanConfigDS complianceConfigDS.DataStore, integrationDS complianceIntegrationDS.DataStore, profileDS profileDatastore.DataStore, ruleDS complianceRuleDS.DataStore, scanDS complianceScanDS.DataStore) Service {
	return &serviceImpl{
		complianceResultsDS: complianceResultsDS,
		scanConfigDS:        scanConfigDS,
		integrationDS:       integrationDS,
		profileDS:           profileDS,
		ruleDS:              ruleDS,
		scanDS:              scanDS,
	}
}

type serviceImpl struct {
	v2.UnimplementedComplianceResultsServiceServer

	complianceResultsDS complianceDS.DataStore
	scanConfigDS        complianceConfigDS.DataStore
	integrationDS       complianceIntegrationDS.DataStore
	profileDS           profileDatastore.DataStore
	ruleDS              complianceRuleDS.DataStore
	scanDS              complianceScanDS.DataStore
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
func (s *serviceImpl) GetComplianceScanResults(ctx context.Context, query *v2.RawQuery) (*v2.ListComplianceResultsResponse, error) {
	// Fill in Query.
	parsedQuery, err := search.ParseQuery(query.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to parse query %v", err)
	}

	countQuery := parsedQuery.Clone()

	// Fill in pagination.
	paginated.FillPaginationV2(parsedQuery, query.GetPagination(), maxPaginationLimit)

	return s.searchComplianceCheckResults(ctx, parsedQuery, countQuery)
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

	// Check the Compliance Scan object to get the scan time.
	scanQuery := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorScanRef, scanResult.GetScanRefId()).
		ProtoQuery()
	scans, err := s.scanDS.SearchScans(ctx, scanQuery)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to retrieve scan data for result %q", req.GetId())
	}
	if len(scans) == 0 {
		return nil, errors.Errorf("Unable to retrieve scan data for result %q", req.GetId())
	}

	var lastScanTime *types.Timestamp
	for _, scan := range scans {
		if types.CompareTimestamps(scan.LastExecutedTime, lastScanTime) > 0 {
			lastScanTime = scan.LastExecutedTime
		}
	}

	return storagetov2.ComplianceV2CheckResult(scanResult, lastScanTime), nil
}

// GetComplianceScanConfigurationResults retrieves the most recent compliance operator scan results for the specified query
// TODO(ROX-20333):  the most recent portion will come when this ticket is worked once everything is wired up so we can tell
// what the latest scan is.
func (s *serviceImpl) GetComplianceScanConfigurationResults(ctx context.Context, request *v2.ComplianceScanResultsRequest) (*v2.ListComplianceResultsResponse, error) {
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

	countQuery := parsedQuery.Clone()

	// Fill in pagination.
	paginated.FillPaginationV2(parsedQuery, request.GetQuery().GetPagination(), maxPaginationLimit)

	return s.searchComplianceCheckResults(ctx, parsedQuery, countQuery)
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

	// Lookup the scans to get the last scan time
	clusterLastScan := make(map[string]*types.Timestamp, len(scanResults))
	for _, result := range scanResults { // Check the Compliance Scan object to get the scan time.
		lastExecutedTime, err := utils.GetLastScanTime(ctx, result.ClusterId, request.GetProfileName(), s.scanDS)
		if err != nil {
			return nil, err
		}
		clusterLastScan[result.ClusterId] = lastExecutedTime
	}

	resultCount, err := s.complianceResultsDS.CountCheckResults(ctx, countQuery)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance scan results count for query %v", parsedQuery)
	}

	return &v2.ListComplianceCheckClusterResponse{
		CheckResults: storagetov2.ComplianceV2CheckClusterResults(scanResults, clusterLastScan),
		ProfileName:  request.GetProfileName(),
		CheckName:    request.GetCheckName(),
		TotalCount:   int32(resultCount),
	}, nil
}

// GetComplianceProfileClusterResults retrieves check results for a specific profile on a specific cluster
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

	// Check the Compliance Scan object to get the scan time.
	lastExecutedTime, err := utils.GetLastScanTime(ctx, request.GetClusterId(), request.GetProfileName(), s.scanDS)
	if err != nil {
		return nil, err
	}

	return &v2.ListComplianceCheckResultResponse{
		CheckResults: storagetov2.ComplianceV2CheckResults(scanResults, checkToRule),
		ProfileName:  request.GetProfileName(),
		ClusterId:    request.GetClusterId(),
		TotalCount:   int32(resultCount),
		LastScanTime: lastExecutedTime,
	}, nil
}

func (s *serviceImpl) GetComplianceProfileCheckDetails(ctx context.Context, request *v2.ComplianceCheckDetailRequest) (*v2.ComplianceClusterCheckStatus, error) {
	if request.GetProfileName() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Profile name is required")
	}
	if request.GetCheckName() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Check name is required")
	}

	// Fill in Query.
	parsedQuery, err := search.ParseQuery(request.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to parse query %v", err)
	}

	// Add the profile and check name to the query
	parsedQuery = search.ConjunctionQuery(
		search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorProfileName, request.GetProfileName()).
			AddExactMatches(search.ComplianceOperatorCheckName, request.GetCheckName()).
			ProtoQuery(),
		parsedQuery,
	)

	scanResults, err := s.complianceResultsDS.SearchComplianceCheckResults(ctx, parsedQuery)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance scan results for query %v", parsedQuery)
	}

	return storagetov2.ComplianceV2SpecificCheckResult(scanResults, request.GetCheckName()), nil
}

func (s *serviceImpl) searchComplianceCheckResults(ctx context.Context, parsedQuery *v1.Query, countQuery *v1.Query) (*v2.ListComplianceResultsResponse, error) {
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

	count, err := s.complianceResultsDS.CountCheckResults(ctx, countQuery)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve count of compliance scan results for query %v", parsedQuery)
	}

	return &v2.ListComplianceResultsResponse{
		ScanResults: storagetov2.ComplianceV2CheckData(scanResults, checkToRule),
		TotalCount:  int32(count),
	}, nil
}
