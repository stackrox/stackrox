package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	benchmarksDS "github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/datastore"
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
	"github.com/stackrox/rox/pkg/logging"
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
			v2.ComplianceResultsService_GetComplianceScanResults_FullMethodName,
			v2.ComplianceResultsService_GetComplianceScanCheckResult_FullMethodName,
			v2.ComplianceResultsService_GetComplianceScanConfigurationResults_FullMethodName,
			v2.ComplianceResultsService_GetComplianceProfileResults_FullMethodName,
			v2.ComplianceResultsService_GetComplianceProfileCheckResult_FullMethodName,
			v2.ComplianceResultsService_GetComplianceProfileClusterResults_FullMethodName,
			v2.ComplianceResultsService_GetComplianceProfileCheckDetails_FullMethodName,
		},
	})

	log = logging.LoggerForModule()
)

// New returns a service object for registering with grpc.
func New(complianceResultsDS complianceDS.DataStore, scanConfigDS complianceConfigDS.DataStore, integrationDS complianceIntegrationDS.DataStore, profileDS profileDatastore.DataStore, ruleDS complianceRuleDS.DataStore, scanDS complianceScanDS.DataStore, benchmarkDS benchmarksDS.DataStore) Service {
	return &serviceImpl{
		complianceResultsDS: complianceResultsDS,
		scanConfigDS:        scanConfigDS,
		integrationDS:       integrationDS,
		profileDS:           profileDS,
		ruleDS:              ruleDS,
		scanDS:              scanDS,
		benchmarkDS:         benchmarkDS,
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
	benchmarkDS         benchmarksDS.DataStore
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

	countQuery := parsedQuery.CloneVT()

	// Fill in pagination.
	paginated.FillPaginationV2(parsedQuery, query.GetPagination(), maxPaginationLimit)

	return s.searchComplianceCheckResults(ctx, parsedQuery, countQuery)
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
	scanRefQuery := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorScanRef, scanResult.GetScanRefId()).
		ProtoQuery()
	scans, err := s.scanDS.SearchScans(ctx, scanRefQuery)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to retrieve scan data for result %q", req.GetId())
	}
	if len(scans) == 0 {
		return nil, errors.Errorf("Unable to find scan data for result %q", req.GetId())
	}

	var lastScanTime *types.Timestamp
	for _, scan := range scans {
		if types.CompareTimestamps(scan.LastExecutedTime, lastScanTime) > 0 {
			lastScanTime = scan.LastExecutedTime
		}
	}

	// Check the Profile so we can get the controls.
	profiles, err := s.profileDS.SearchProfiles(ctx, scanRefQuery)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to retrieve profiles for result %q", req.GetId())
	}
	if len(profiles) == 0 {
		return nil, errors.Errorf("Unable to find profiles for result %q", req.GetId())
	}

	ruleNames := make([]string, 0, 1)
	rules, err := s.ruleDS.SearchRules(ctx, search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorRuleRef, scanResult.GetRuleRefId()).ProtoQuery())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance rule for result %q", req.GetId())
	}
	if len(rules) != 1 {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to process compliance rule for result %q", req.GetId())
	}
	ruleNames = append(ruleNames, rules[0].GetName())

	controls, err := utils.GetControlsForScanResults(ctx, s.ruleDS, ruleNames, profiles[0].GetName(), s.benchmarkDS)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve controls for result %q", req.GetId())
	}

	return storagetov2.ComplianceV2CheckResult(scanResult, lastScanTime, ruleNames[0], controls), nil
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

	countQuery := parsedQuery.CloneVT()

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
	countQuery := parsedQuery.CloneVT()

	// Fill in pagination.
	paginated.FillPaginationV2(parsedQuery, request.GetQuery().GetPagination(), maxPaginationLimit)

	scanResults, err := s.complianceResultsDS.ComplianceProfileResults(ctx, parsedQuery)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to retrieve compliance profile scan stats for %+v", request)
	}

	ruleNames := make([]string, 0, len(scanResults))
	for _, result := range scanResults {
		ruleNames = append(ruleNames, result.RuleName)
	}

	count, err := s.complianceResultsDS.CountByField(ctx, countQuery, search.ComplianceOperatorCheckName)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance scan results count for query %v", request)
	}

	controls, err := utils.GetControlsForScanResults(ctx, s.ruleDS, ruleNames, request.GetProfileName(), s.benchmarkDS)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve controls for compliance scan results %v", request)
	}

	return &v2.ListComplianceProfileResults{
		ProfileResults: storagetov2.ComplianceV2ProfileResults(scanResults, controls),
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
	countQuery := parsedQuery.CloneVT()

	// Fill in pagination.
	paginated.FillPaginationV2(parsedQuery, request.GetQuery().GetPagination(), maxPaginationLimit)

	scanResults, err := s.complianceResultsDS.SearchComplianceCheckResults(ctx, parsedQuery)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance scan results for query %v", parsedQuery)
	}

	// Lookup the scans to get the last scan time
	clusterLastScan := make(map[string]*types.Timestamp, len(scanResults))

	// This is a single check which has results across clusters.  So there will be one single underlying rule.
	ruleNames := make([]string, 0, 1)
	for _, result := range scanResults { // Check the Compliance Scan object to get the scan time.
		lastExecutedTime, err := utils.GetLastScanTime(ctx, result.ClusterId, request.GetProfileName(), s.scanDS)
		if err != nil {
			return nil, err
		}
		clusterLastScan[result.ClusterId] = lastExecutedTime

		if len(ruleNames) == 0 {
			rules, err := s.ruleDS.SearchRules(ctx, search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorRuleRef, result.GetRuleRefId()).ProtoQuery())
			if err != nil {
				return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance rule for query %v", parsedQuery)
			}
			if len(rules) != 1 {
				return nil, errors.Wrapf(errox.InvalidArgs, "Unable to process compliance rule for query %v", parsedQuery)
			}
			ruleNames = append(ruleNames, rules[0].GetName())
		}
	}

	resultCount, err := s.complianceResultsDS.CountCheckResults(ctx, countQuery)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance scan results count for query %v", parsedQuery)
	}

	controls, err := utils.GetControlsForScanResults(ctx, s.ruleDS, ruleNames, request.GetProfileName(), s.benchmarkDS)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve controls for compliance scan results %v", request)
	}

	var convertedControls []*v2.ComplianceControl
	if len(ruleNames) == 1 {
		convertedControls = storagetov2.GetControls(ruleNames[0], controls)
	}

	return &v2.ListComplianceCheckClusterResponse{
		CheckResults: storagetov2.ComplianceV2CheckClusterResults(scanResults, clusterLastScan),
		ProfileName:  request.GetProfileName(),
		CheckName:    request.GetCheckName(),
		TotalCount:   int32(resultCount),
		Controls:     convertedControls,
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

	countQuery := parsedQuery.CloneVT()

	// Fill in pagination.
	paginated.FillPaginationV2(parsedQuery, request.GetQuery().GetPagination(), maxPaginationLimit)

	scanResults, err := s.complianceResultsDS.SearchComplianceCheckResults(ctx, parsedQuery)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance scan results for query %v", parsedQuery)
	}

	checkToRule := make(map[string]string, len(scanResults))
	ruleNames := make([]string, 0, len(scanResults))
	for _, result := range scanResults {
		rules, err := s.ruleDS.SearchRules(ctx, search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorRuleRef, result.GetRuleRefId()).ProtoQuery())
		if err != nil {
			return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance rule for query %v", parsedQuery)
		}
		if len(rules) != 1 {
			return nil, errors.Wrapf(errox.InvalidArgs, "Unable to process compliance rule for query %v", parsedQuery)
		}
		checkToRule[result.GetRuleRefId()] = rules[0].GetName()
		ruleNames = append(ruleNames, rules[0].GetName())
	}

	controls, err := utils.GetControlsForScanResults(ctx, s.ruleDS, ruleNames, request.GetProfileName(), s.benchmarkDS)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve controls for compliance scan results %v", request)
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
		CheckResults: storagetov2.ComplianceV2CheckResults(scanResults, checkToRule, controls),
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
	if len(scanResults) == 0 {
		return nil, nil
	}

	// The goal of this API is to return the details of a check result in a normalized manner.  A view of the check
	// result details that should match across the clusters as the profile version is part of the check result name.
	// Since the data is denormalized and our goal is to look up the rule that matches the result, we only need to
	// grab the rule from a single result as all the rules across the clusters will match the same thing.  As the name
	// of the check result contains the profile version in it, the rule the result maps to will be the same
	// across all clusters.
	rules, err := s.ruleDS.SearchRules(ctx, search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorRuleRef, scanResults[0].GetRuleRefId()).ProtoQuery())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance rule for query %v", parsedQuery)
	}
	// Since we are using the `RuleRefId` of the first result to find the underlying rule, there can only be 1 rule.
	// The first result will have a specific cluster and the `RuleRefId` is built from rule_name and cluster_id so to
	// get the rule name associated with this result we only need to check one as the profile version is in the check
	// result name and as such should match across clusters.  Later it would be good to abstract some of this to the
	// datastore by getting distinct information.
	if len(rules) != 1 {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to process compliance rule for query %v", parsedQuery)
	}

	scanRefQuery := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorScanRef, scanResults[0].GetScanRefId()).
		ProtoQuery()
	profiles, err := s.profileDS.SearchProfiles(ctx, scanRefQuery)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to retrieve profiles for result %v", parsedQuery)
	}
	var convertedControls []*v2.ComplianceControl
	if len(profiles) == 0 {
		// TODO(ROX-22362): implement tailored profiles
		log.Warnf("Unable to find profiles for result %v.  It is possible results match a tailored profile which have not been implemented in Compliance V2", parsedQuery)
	} else {
		controls, err := utils.GetControlsForScanResults(ctx, s.ruleDS, []string{rules[0].GetName()}, profiles[0].GetName(), s.benchmarkDS)
		if err != nil {
			return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve controls for compliance scan results %v", parsedQuery)
		}
		convertedControls = storagetov2.GetControls(rules[0].GetName(), controls)
	}

	return storagetov2.ComplianceV2SpecificCheckResult(scanResults, request.GetCheckName(), convertedControls), nil
}

func (s *serviceImpl) searchComplianceCheckResults(ctx context.Context, parsedQuery *v1.Query, countQuery *v1.Query) (*v2.ListComplianceResultsResponse, error) {
	scanResults, err := s.complianceResultsDS.SearchComplianceCheckResults(ctx, parsedQuery)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance scan results for query %v", parsedQuery)
	}

	checkToRule := make(map[string]string, len(scanResults))
	checkToControls := make(map[string][]*complianceRuleDS.ControlResult, len(scanResults))
	// Cache profiles for scan ref id so we don't have to look them up each time.
	profileCache := make(map[string]string, len(scanResults))
	for _, result := range scanResults {
		rules, err := s.ruleDS.SearchRules(ctx, search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorRuleRef, result.GetRuleRefId()).ProtoQuery())
		if err != nil {
			return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance rule for query %v", parsedQuery)
		}
		if len(rules) != 1 {
			return nil, errors.Wrapf(errox.InvalidArgs, "Unable to process compliance rule for query %v", parsedQuery)
		}
		checkToRule[result.GetRuleRefId()] = rules[0].GetName()

		// Check the Profile so we can get the controls.
		profileName, found := profileCache[result.GetScanRefId()]
		if !found {
			scanRefQuery := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorScanRef, result.GetScanRefId()).
				ProtoQuery()
			profiles, err := s.profileDS.SearchProfiles(ctx, scanRefQuery)
			if err != nil {
				return nil, errors.Wrapf(err, "Unable to retrieve profiles for result %v", parsedQuery)
			}
			if len(profiles) == 0 {
				// TODO(ROX-22362): implement tailored profiles
				log.Warnf("Unable to find profiles for result %v.  It is possible results match a tailored profile which have not been implemented in Compliance V2", parsedQuery)
				continue
			}
			profileName = profiles[0].GetName()
			profileCache[result.GetScanRefId()] = profileName
		}

		if _, found := checkToControls[result.GetCheckName()]; !found {
			controls, err := utils.GetControlsForScanResults(ctx, s.ruleDS, []string{rules[0].GetName()}, profileName, s.benchmarkDS)
			if err != nil {
				return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve controls for compliance scan results %v", parsedQuery)
			}

			checkToControls[result.GetCheckName()] = controls
		}
	}

	count, err := s.complianceResultsDS.CountCheckResults(ctx, countQuery)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve count of compliance scan results for query %v", parsedQuery)
	}

	return &v2.ListComplianceResultsResponse{
		ScanResults: storagetov2.ComplianceV2CheckData(scanResults, checkToRule, checkToControls),
		TotalCount:  int32(count),
	}, nil
}
