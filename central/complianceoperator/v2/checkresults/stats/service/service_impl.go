package service

import (
	"context"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/complianceoperator/v2/benchmark"
	complianceDS "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	"github.com/stackrox/rox/central/complianceoperator/v2/checkresults/utils"
	compliancedata "github.com/stackrox/rox/central/complianceoperator/v2/compliancedata"
	complianceIntegrationDS "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore"
	profileDatastore "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore"
	ruleDS "github.com/stackrox/rox/central/complianceoperator/v2/rules/datastore"
	complianceConfigDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	complianceScanDS "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	"github.com/stackrox/rox/central/convert/storagetov2"
	v1 "github.com/stackrox/rox/generated/api/v1"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/logging"
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
		user.With(permissions.View(resources.Compliance), permissions.View(resources.Cluster)): {
			v2.ComplianceResultsStatsService_GetComplianceProfileStats_FullMethodName,
			v2.ComplianceResultsStatsService_GetComplianceProfilesStats_FullMethodName,
			v2.ComplianceResultsStatsService_GetComplianceProfilesClusterStats_FullMethodName,
			v2.ComplianceResultsStatsService_GetComplianceClusterScanStats_FullMethodName,
			v2.ComplianceResultsStatsService_GetComplianceOverallClusterStats_FullMethodName,
			v2.ComplianceResultsStatsService_GetComplianceClusterStats_FullMethodName,
			v2.ComplianceResultsStatsService_GetComplianceProfileCheckStats_FullMethodName,
		},
	})

	log = logging.LoggerForModule()
)

// New returns a service object for registering with grpc.
func New(complianceResultsDS complianceDS.DataStore, scanConfigDS complianceConfigDS.DataStore, integrationDS complianceIntegrationDS.DataStore, profileDS profileDatastore.DataStore, scanDS complianceScanDS.DataStore, ruleDS ruleDS.DataStore, clusterDS clusterDatastore.DataStore) Service {
	return &serviceImpl{
		complianceResultsDS: complianceResultsDS,
		scanConfigDS:        scanConfigDS,
		integrationDS:       integrationDS,
		profileDS:           profileDS,
		scanDS:              scanDS,
		ruleDS:              ruleDS,
		clusterDS:           clusterDS,
	}
}

type serviceImpl struct {
	v2.UnimplementedComplianceResultsStatsServiceServer

	complianceResultsDS complianceDS.DataStore
	scanConfigDS        complianceConfigDS.DataStore
	integrationDS       complianceIntegrationDS.DataStore
	profileDS           profileDatastore.DataStore
	scanDS              complianceScanDS.DataStore
	ruleDS              ruleDS.DataStore
	clusterDS           clusterDatastore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v2.RegisterComplianceResultsStatsServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v2.RegisterComplianceResultsStatsServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
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
	countQuery := parsedQuery.CloneVT()

	// Fill in pagination.
	paginated.FillPaginationV2(parsedQuery, request.GetQuery().GetPagination(), maxPaginationLimit)

	profileStats, count, err := s.getProfileStats(ctx, parsedQuery, countQuery)
	return &v2.ListComplianceProfileScanStatsResponse{
		ScanStats:  profileStats,
		TotalCount: int32(count),
	}, err
}

// GetComplianceProfilesStats lists current scan stats grouped by profile
func (s *serviceImpl) GetComplianceProfilesStats(ctx context.Context, query *v2.RawQuery) (*v2.ListComplianceProfileScanStatsResponse, error) {
	// Fill in Query.
	parsedQuery, err := search.ParseQuery(query.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to parse query %v", err)
	}

	// To get total count, need the parsed query without the paging.
	countQuery := parsedQuery.CloneVT()

	// Fill in pagination.
	paginated.FillPaginationV2(parsedQuery, query.GetPagination(), maxPaginationLimit)

	profileStats, count, err := s.getProfileStats(ctx, parsedQuery, countQuery)
	return &v2.ListComplianceProfileScanStatsResponse{
		ScanStats:  profileStats,
		TotalCount: int32(count),
	}, err
}

// GetComplianceProfilesClusterStats lists current scan stats grouped by profile
func (s *serviceImpl) GetComplianceProfilesClusterStats(ctx context.Context, request *v2.ComplianceScanClusterRequest) (*v2.ListComplianceClusterProfileStatsResponse, error) {
	if request.GetClusterId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Cluster ID is required")
	}

	clusterName, found, err := s.clusterDS.GetClusterName(ctx, request.GetClusterId())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable retrieve cluster %v", err)
	}
	if !found {
		return nil, errors.Wrapf(errox.InvalidArgs, "Cluster %q does not exist", request.GetClusterId())
	}

	// Fill in Query.
	parsedQuery, err := search.ParseQuery(request.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to parse query %v", err)
	}

	// Add the cluster id as an exact match
	parsedQuery = search.ConjunctionQuery(
		search.NewQueryBuilder().AddExactMatches(search.ClusterID, request.GetClusterId()).ProtoQuery(),
		parsedQuery,
	)

	// To get total count, need the parsed query without the paging.
	countQuery := parsedQuery.CloneVT()

	// Fill in pagination.
	paginated.FillPaginationV2(parsedQuery, request.GetQuery().GetPagination(), maxPaginationLimit)

	profileStats, count, err := s.getProfileStats(ctx, parsedQuery, countQuery)
	return &v2.ListComplianceClusterProfileStatsResponse{
		ScanStats:   profileStats,
		ClusterId:   request.GetClusterId(),
		ClusterName: clusterName,
		TotalCount:  int32(count),
	}, err
}

func (s *serviceImpl) getProfileStats(ctx context.Context, parsedQuery *v1.Query, countQuery *v1.Query) ([]*v2.ComplianceProfileScanStats, int, error) {
	scanResults, err := s.complianceResultsDS.ComplianceProfileResultStats(ctx, parsedQuery)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "Unable to retrieve compliance profile scan stats for %+v", parsedQuery)
	}

	count, err := s.complianceResultsDS.CountByField(ctx, countQuery, search.ComplianceOperatorProfileName)
	if err != nil {
		return nil, 0, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance scan results count for request %v", countQuery)
	}
	profileMap := map[string]*storage.ComplianceOperatorProfileV2{}
	profileBenchmarksMap := map[string][]*storage.ComplianceOperatorBenchmarkV2{}
	for _, scan := range scanResults {
		profileResults, err := s.profileDS.SearchProfiles(ctx, search.NewQueryBuilder().
			AddExactMatches(search.ComplianceOperatorProfileName, scan.ProfileName).ProtoQuery())
		if err != nil {
			return nil, 0, errors.Wrap(err, "Unable to retrieve compliance profile")
		}
		if len(profileResults) == 0 {
			return nil, 0, errors.Errorf("Unable to retrieve compliance profile for %s", scan.ProfileName)
		}

		profileMap[scan.ProfileName] = profileResults[0]

		// Get the benchmarks
		if _, found := profileBenchmarksMap[scan.ProfileName]; !found {
			profileBenchmark, err := benchmark.GetBenchmarkFromProfile(profileMap[scan.ProfileName])
			if err != nil {
				return nil, 0, errors.Wrapf(err, "failed to retrieve benchmarks for profile %q.", scan.ProfileName)
			}
			profileBenchmarksMap[scan.ProfileName] = []*storage.ComplianceOperatorBenchmarkV2{profileBenchmark}
		}
	}

	return storagetov2.ComplianceV2ProfileStats(scanResults, profileMap, profileBenchmarksMap), count, nil
}

// GetComplianceClusterScanStats lists current scan stats for a cluster for each scan configuration
func (s *serviceImpl) GetComplianceClusterScanStats(ctx context.Context, request *v2.ComplianceScanClusterRequest) (*v2.ListComplianceClusterScanStatsResponse, error) {
	if request.GetClusterId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Cluster ID is required")
	}

	// Fill in Query.
	parsedQuery, err := search.ParseQuery(request.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to parse query %v", err)
	}

	// Add the cluster id as an exact match
	parsedQuery = search.ConjunctionQuery(
		search.NewQueryBuilder().AddExactMatches(search.ClusterID, request.GetClusterId()).ProtoQuery(),
		parsedQuery,
	)

	countQuery := parsedQuery.CloneVT()

	// Fill in pagination.
	paginated.FillPaginationV2(parsedQuery, request.GetQuery().GetPagination(), maxPaginationLimit)

	scanResults, err := s.complianceResultsDS.ComplianceCheckResultStats(ctx, parsedQuery)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance cluster scan stats for request %v", request)
	}

	// Need to look up the scan config IDs to return with the results.
	// outdatedEnabled captures the flag once so the error semantics below and the
	// data-state computation stay consistent within this call.
	outdatedEnabled := features.ComplianceSurfaceStaleData.Enabled()
	scanConfigToIDs := make(map[string]string, len(scanResults))
	var loadedConfigs []*storage.ComplianceOperatorScanConfigurationV2
	for _, result := range scanResults {
		if _, found := scanConfigToIDs[result.ScanConfigName]; !found {
			config, err := s.scanConfigDS.GetScanConfigurationByName(ctx, result.ScanConfigName)
			if err != nil {
				// With the flag off keep the original hard error so behavior is
				// unchanged. With the flag on a missing config must not fail the
				// endpoint: its data_state simply resolves to UNKNOWN.
				if !outdatedEnabled {
					return nil, errors.Errorf("Unable to retrieve valid compliance scan configuration for results from %v", request)
				}
				log.Warnf("compliance outdated: cannot resolve scan config %q: %v; its data_state degrades to UNKNOWN", result.ScanConfigName, err)
				scanConfigToIDs[result.ScanConfigName] = "" // unresolvable; data_state → UNKNOWN
				continue
			}
			// GetScanConfigurationByName returns (nil, nil) for a not-found config.
			// GetId() is nil-safe (""), preserving the pre-feature scanConfigToIDs
			// behavior, but a nil config must not enter the resolver's config list
			// (matches the checkresults service's cfg == nil guard).
			scanConfigToIDs[result.ScanConfigName] = config.GetId()
			if config != nil {
				loadedConfigs = append(loadedConfigs, config)
			}
		}
	}

	// Compute data state per scan config (feature-flag gated; flag off ⇒ empty
	// map ⇒ UNKNOWN and no extra datastore query).
	configDataStates := make(map[string]v2.ComplianceDataState, len(scanConfigToIDs))
	if outdatedEnabled && len(loadedConfigs) > 0 {
		// Use countQuery (unpaginated) for consistency with the sibling MIN sites;
		// the datastore strips pagination from aggregates so this is not a
		// correctness change, only consistency.
		minTimes, err := s.complianceResultsDS.MinLastStartedTimeByConfigCluster(ctx, countQuery)
		if err == nil {
			resolver := compliancedata.NewConfigResolver(loadedConfigs, time.Now().UTC())
			for _, mt := range minTimes {
				configDataStates[mt.ScanConfigName] = resolver.ResolveGroupedMin(mt.ScanConfigName, mt.MinLastStarted).ToProto()
			}
		} else {
			log.Warnf("compliance outdated: failed to get min times for cluster scan stats: %v; all data_states degrade to UNKNOWN", err)
		}
	}

	count, err := s.complianceResultsDS.CountByField(ctx, countQuery, search.ComplianceOperatorScanConfigName)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance scan results count for request %v", request)
	}

	return &v2.ListComplianceClusterScanStatsResponse{
		ScanStats:  storagetov2.ComplianceV2ClusterStats(scanResults, scanConfigToIDs, configDataStates),
		TotalCount: int32(count),
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
	countQuery := parsedQuery.CloneVT()

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

	// Outdated-data detection is feature-flag gated; with the flag off the fields
	// stay UNKNOWN / 0 and no extra datastore query runs (prior behavior).
	var clusterDataStates map[string]v2.ComplianceDataState
	var outdatedCount int32
	if features.ComplianceSurfaceStaleData.Enabled() {
		// Use countQuery (unpaginated) for the banner signal, so the count covers the
		// full filtered scope, not just the current page.
		clusterDataStates, err = s.computeClusterDataStates(ctx, countQuery)
		if err != nil {
			log.Warnf("compliance outdated: failed to compute cluster data states: %v; outdated banner hidden", err)
			clusterDataStates = make(map[string]v2.ComplianceDataState)
		}
		for _, state := range clusterDataStates {
			if state == v2.ComplianceDataState_COMPLIANCE_DATA_STATE_OUTDATED {
				outdatedCount++
			}
		}
	}

	return &v2.ListComplianceClusterOverallStatsResponse{
		ScanStats:            storagetov2.ComplianceV2ClusterOverallStats(scanResults, clusterErrors, clusterDataStates),
		TotalCount:           int32(count),
		OutdatedClusterCount: outdatedCount,
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
	countQuery := parsedQuery.CloneVT()

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
		// Get the integrations if we can.  If we cannot, it could be an externally configured
		// scan, and thus we will not have a matching integration.
		integrations, err := s.integrationDS.GetComplianceIntegrationByCluster(ctx, result.ClusterID)
		if err != nil {
			return nil, errors.Wrapf(err, "Unable to retrieve configuration for cluster %q", result.ClusterID)
		}
		if len(integrations) == 1 {
			clusterErrors[result.ClusterID] = integrations[0].GetStatusErrors()
		} else if len(integrations) < 1 {
			log.Warnf("Unable to detect a compliance operator integration for cluster %q", result.ClusterID)
			clusterErrors[result.ClusterID] = []string{"Unable to detect a compliance operator integration"}
		} else {
			log.Warnf("Detected multiple compliance operator integrations for cluster %q", result.ClusterID)
			clusterErrors[result.ClusterID] = []string{"Detected multiple compliance operator integrations"}
		}
	}

	// Outdated-data detection is feature-flag gated; with the flag off the fields
	// stay UNKNOWN / 0 and no extra datastore query runs (prior behavior).
	var clusterDataStates map[string]v2.ComplianceDataState
	var outdatedCount int32
	if features.ComplianceSurfaceStaleData.Enabled() {
		// Use countQuery (unpaginated) for the banner signal, so the count covers the
		// full filtered scope, not just the current page.
		clusterDataStates, err = s.computeClusterDataStates(ctx, countQuery)
		if err != nil {
			log.Warnf("compliance outdated: failed to compute cluster data states: %v; outdated banner hidden", err)
			clusterDataStates = make(map[string]v2.ComplianceDataState)
		}
		for _, state := range clusterDataStates {
			if state == v2.ComplianceDataState_COMPLIANCE_DATA_STATE_OUTDATED {
				outdatedCount++
			}
		}
	}

	return &v2.ListComplianceClusterOverallStatsResponse{
		ScanStats:            storagetov2.ComplianceV2ClusterOverallStats(scanResults, clusterErrors, clusterDataStates),
		TotalCount:           int32(count),
		OutdatedClusterCount: outdatedCount,
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

	ruleNames := make([]string, 0, len(scanResults))
	for _, result := range scanResults {
		ruleNames = append(ruleNames, result.RuleName)
	}

	controls, err := utils.GetControlsForScanResults(ctx, s.ruleDS, ruleNames, request.GetProfileName())
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to retrieve controls for compliance profile check stats for %+v", request)
	}

	return &v2.ListComplianceProfileResults{
		// Single-check stats view (not the Checks tab); no per-check freshness rollup here.
		ProfileResults: storagetov2.ComplianceV2ProfileResults(scanResults, controls, nil),
		ProfileName:    request.GetProfileName(),
		TotalCount:     int32(1),
	}, nil
}

// computeClusterDataStates returns a map cluster_id → rolled-up freshness state
// over the (scan_config_name, cluster_id) MIN(last_started_time) aggregate.
func (s *serviceImpl) computeClusterDataStates(ctx context.Context, query *v1.Query) (map[string]v2.ComplianceDataState, error) {
	minTimes, err := s.complianceResultsDS.MinLastStartedTimeByConfigCluster(ctx, query)
	if err != nil {
		return nil, err
	}

	configNames := make(map[string]struct{})
	for _, mt := range minTimes {
		configNames[mt.ScanConfigName] = struct{}{}
	}
	resolver := s.buildConfigResolver(ctx, configNames)

	perClusterStates := make(map[string][]compliancedata.State)
	for _, mt := range minTimes {
		state := resolver.ResolveGroupedMin(mt.ScanConfigName, mt.MinLastStarted)
		perClusterStates[mt.ClusterID] = append(perClusterStates[mt.ClusterID], state)
	}

	result := make(map[string]v2.ComplianceDataState, len(perClusterStates))
	for clusterID, states := range perClusterStates {
		result[clusterID] = compliancedata.RollupState(states...).ToProto()
	}
	return result, nil
}

// buildConfigResolver loads the named scan configs in a SINGLE datastore query
// (OR-ing the names) instead of a per-name point read, and returns a resolver
// anchored at the current time. Mirrors the checkresults service helper.
// Best-effort: on a load error, or for names the query does not return, the
// affected configs resolve to UNKNOWN rather than failing the page.
func (s *serviceImpl) buildConfigResolver(ctx context.Context, configNames map[string]struct{}) *compliancedata.ConfigResolver {
	now := time.Now().UTC()
	if len(configNames) == 0 {
		return compliancedata.NewConfigResolver(nil, now)
	}

	names := make([]string, 0, len(configNames))
	for name := range configNames {
		names = append(names, name)
	}

	configs, err := s.scanConfigDS.GetScanConfigurations(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorScanConfigName, names...).ProtoQuery())
	if err != nil {
		// Best-effort observability: this degrades the affected clusters to
		// UNKNOWN (silently dropping any OUTDATED signal), so log it with the
		// config names for diagnosis.
		log.Warnf("compliance outdated: cannot load scan configs %v: %v; affected clusters degrade to UNKNOWN", names, err)
	}
	return compliancedata.NewConfigResolver(configs, now)
}
