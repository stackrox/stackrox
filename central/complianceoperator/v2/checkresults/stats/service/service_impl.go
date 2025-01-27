package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	benchmarksDS "github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/datastore"
	complianceDS "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	"github.com/stackrox/rox/central/complianceoperator/v2/checkresults/utils"
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
		user.With(permissions.View(resources.Compliance)): {
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
func New(complianceResultsDS complianceDS.DataStore, scanConfigDS complianceConfigDS.DataStore, integrationDS complianceIntegrationDS.DataStore, profileDS profileDatastore.DataStore, scanDS complianceScanDS.DataStore, benchmarkDS benchmarksDS.DataStore, ruleDS ruleDS.DataStore, clusterDS clusterDatastore.DataStore) Service {
	return &serviceImpl{
		complianceResultsDS: complianceResultsDS,
		scanConfigDS:        scanConfigDS,
		integrationDS:       integrationDS,
		profileDS:           profileDS,
		scanDS:              scanDS,
		ruleDS:              ruleDS,
		clusterDS:           clusterDS,
		benchmarkDS:         benchmarkDS,
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
	benchmarkDS         benchmarksDS.DataStore
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
			benchmarks, err := s.benchmarkDS.GetBenchmarksByProfileName(ctx, scan.ProfileName)
			if err != nil {
				return nil, 0, errors.Wrapf(err, "failed to retrieve benchmarks for profile %q.", scan.ProfileName)
			}
			profileBenchmarksMap[scan.ProfileName] = benchmarks
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
	scanConfigToIDs := make(map[string]string, len(scanResults))
	for _, result := range scanResults {
		if _, found := scanConfigToIDs[result.ScanConfigName]; !found {
			config, err := s.scanConfigDS.GetScanConfigurationByName(ctx, result.ScanConfigName)
			if err != nil {
				return nil, errors.Errorf("Unable to retrieve valid compliance scan configuration for results from %v", request)
			}
			scanConfigToIDs[result.ScanConfigName] = config.GetId()
		}
	}

	count, err := s.complianceResultsDS.CountByField(ctx, countQuery, search.ComplianceOperatorScanConfigName)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance scan results count for request %v", request)
	}

	return &v2.ListComplianceClusterScanStatsResponse{
		ScanStats:  storagetov2.ComplianceV2ClusterStats(scanResults, scanConfigToIDs),
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

	return &v2.ListComplianceClusterOverallStatsResponse{
		ScanStats:  storagetov2.ComplianceV2ClusterOverallStats(scanResults, clusterErrors),
		TotalCount: int32(count),
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

	controls, err := utils.GetControlsForScanResults(ctx, s.ruleDS, ruleNames, request.GetProfileName(), s.benchmarkDS)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to retrieve controls for compliance profile check stats for %+v", request)
	}

	return &v2.ListComplianceProfileResults{
		ProfileResults: storagetov2.ComplianceV2ProfileResults(scanResults, controls),
		ProfileName:    request.GetProfileName(),
		TotalCount:     int32(1),
	}, nil
}
