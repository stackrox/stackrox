package service

import (
	"context"
	"regexp"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	blobDS "github.com/stackrox/rox/central/blob/datastore"
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	benchmarksDS "github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/datastore"
	"github.com/stackrox/rox/central/complianceoperator/v2/compliancemanager"
	profileDS "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore"
	snapshotDS "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore"
	complianceReportManager "github.com/stackrox/rox/central/complianceoperator/v2/report/manager"
	scanConfigDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	scanSettingBindingsDS "github.com/stackrox/rox/central/complianceoperator/v2/scansettingbindings/datastore"
	suiteDS "github.com/stackrox/rox/central/complianceoperator/v2/suites/datastore"
	"github.com/stackrox/rox/central/convert/storagetov2"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/reports/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	types "github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"google.golang.org/grpc"
	"k8s.io/utils/strings/slices"
)

const (
	maxPaginationLimit = 1000
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Compliance)): {
			v2.ComplianceScanConfigurationService_ListComplianceScanConfigurations_FullMethodName,
			v2.ComplianceScanConfigurationService_GetComplianceScanConfiguration_FullMethodName,
			v2.ComplianceScanConfigurationService_ListComplianceScanConfigProfiles_FullMethodName,
			v2.ComplianceScanConfigurationService_ListComplianceScanConfigClusterProfiles_FullMethodName,
			v2.ComplianceScanConfigurationService_GetReportHistory_FullMethodName,
			v2.ComplianceScanConfigurationService_GetMyReportHistory_FullMethodName,
		},
		user.With(permissions.Modify(resources.Compliance)): {
			v2.ComplianceScanConfigurationService_CreateComplianceScanConfiguration_FullMethodName,
			v2.ComplianceScanConfigurationService_DeleteComplianceScanConfiguration_FullMethodName,
			v2.ComplianceScanConfigurationService_RunComplianceScanConfiguration_FullMethodName,
			v2.ComplianceScanConfigurationService_UpdateComplianceScanConfiguration_FullMethodName,
			v2.ComplianceScanConfigurationService_RunReport_FullMethodName,
			v2.ComplianceScanConfigurationService_DeleteReport_FullMethodName,
		},
	})

	configNameRegexp = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?$`)

	reservedConfigNames = []string{"default", "default-auto-apply"}
)

// New returns a service object for registering with grpc.
func New(scanConfigDS scanConfigDS.DataStore, scanSettingBindingsDS scanSettingBindingsDS.DataStore,
	suiteDS suiteDS.DataStore, manager compliancemanager.Manager, reportManager complianceReportManager.Manager, notifierDS notifierDS.DataStore, profileDS profileDS.DataStore,
	benchmarkDS benchmarksDS.DataStore, clusterDS clusterDatastore.DataStore, snapshotDS snapshotDS.DataStore, blobDS blobDS.Datastore) Service {
	return &serviceImpl{
		scanConfigDS:                    scanConfigDS,
		complianceScanSettingBindingsDS: scanSettingBindingsDS,
		suiteDS:                         suiteDS,
		manager:                         manager,
		reportManager:                   reportManager,
		notifierDS:                      notifierDS,
		profileDS:                       profileDS,
		benchmarkDS:                     benchmarkDS,
		clusterDS:                       clusterDS,
		snapshotDS:                      snapshotDS,
		blobDS:                          blobDS,
	}
}

type serviceImpl struct {
	v2.UnimplementedComplianceScanConfigurationServiceServer

	scanConfigDS                    scanConfigDS.DataStore
	complianceScanSettingBindingsDS scanSettingBindingsDS.DataStore
	suiteDS                         suiteDS.DataStore
	manager                         compliancemanager.Manager
	reportManager                   complianceReportManager.Manager
	notifierDS                      notifierDS.DataStore
	profileDS                       profileDS.DataStore
	benchmarkDS                     benchmarksDS.DataStore
	clusterDS                       clusterDatastore.DataStore
	snapshotDS                      snapshotDS.DataStore
	blobDS                          blobDS.Datastore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v2.RegisterComplianceScanConfigurationServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v2.RegisterComplianceScanConfigurationServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) CreateComplianceScanConfiguration(ctx context.Context, req *v2.ComplianceScanConfiguration) (*v2.ComplianceScanConfiguration, error) {
	if req.GetScanName() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Scan configuration name is required")
	}

	if slices.Contains(reservedConfigNames, strings.ToLower(req.GetScanName())) {
		return nil, errors.Wrapf(errox.InvalidArgs, "Scan configuration name %q cannot be used as it is reserved by the Compliance Operator", req.GetScanName())
	}

	validName := configNameRegexp.MatchString(req.GetScanName())

	if !validName {
		return nil, errors.Wrapf(errox.InvalidArgs, "Scan configuration name %q is not a valid name", req.GetScanName())
	}

	if err := validateScanConfiguration(req); err != nil {
		return nil, err
	}

	// Convert to storage type
	scanConfig := convertV2ScanConfigToStorage(ctx, req)

	// grab clusters
	var clusterIDs []string
	clusterIDs = append(clusterIDs, req.GetClusters()...)

	// Process scan request, config may be updated in the event of errors from sensor.
	scanConfig, err := s.manager.ProcessScanRequest(ctx, scanConfig, clusterIDs)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to process scan config. %v", err)
	}

	return convertStorageScanConfigToV2(ctx, scanConfig, s.scanConfigDS)
}

func (s *serviceImpl) UpdateComplianceScanConfiguration(ctx context.Context, req *v2.ComplianceScanConfiguration) (*v2.Empty, error) {
	if req.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Scan configuration ID is required")
	}

	if err := validateScanConfiguration(req); err != nil {
		return nil, err
	}

	// Convert to storage type
	scanConfig := convertV2ScanConfigToStorage(ctx, req)

	// grab clusters
	var clusterIDs []string
	clusterIDs = append(clusterIDs, req.GetClusters()...)

	// Update scan request, config may be updated in the event of errors from sensor.
	_, err := s.manager.UpdateScanRequest(ctx, scanConfig, clusterIDs)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to process scan config. %v", err)
	}

	return &v2.Empty{}, nil
}

func (s *serviceImpl) DeleteComplianceScanConfiguration(ctx context.Context, req *v2.ResourceByID) (*v2.Empty, error) {
	if req.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Scan configuration ID is required for deletion")
	}
	// Snapshots get deleted with the ScanConfiguration we need to delete the BlobData before
	query := search.NewQueryBuilder().
		AddExactMatches(
			search.ComplianceOperatorScanConfig,
			req.GetId(),
		).
		AddExactMatches(
			search.ComplianceOperatorReportNotificationMethod,
			storage.ComplianceOperatorReportStatus_DOWNLOAD.String(),
		).
		AddExactMatches(
			search.ComplianceOperatorReportState,
			storage.ComplianceOperatorReportStatus_DELIVERED.String(),
			storage.ComplianceOperatorReportStatus_GENERATED.String(),
		).ProtoQuery()
	snapshots, err := s.snapshotDS.SearchSnapshots(ctx, query)
	if err != nil {
		return nil, errors.Wrap(errox.InvariantViolation, "Unable to find the Report Snapshots asociated with the scan config")
	}
	blobCtx := sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Administration)),
	)
	for _, snapshot := range snapshots {
		blobName := common.GetComplianceReportBlobPath(req.GetId(), snapshot.GetReportId())
		if err := s.blobDS.Delete(blobCtx, blobName); err != nil {
			return nil, errors.Wrap(errox.InvariantViolation, "Unable to delete the report asociated with the scan config")
		}
	}

	err = s.manager.DeleteScan(ctx, req.GetId())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to delete scan config: %v", err)
	}

	return &v2.Empty{}, nil
}

func (s *serviceImpl) ListComplianceScanConfigurations(ctx context.Context, query *v2.RawQuery) (*v2.ListComplianceScanConfigurationsResponse, error) {
	// Fill in Query.
	parsedQuery, err := search.ParseQuery(query.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to parse query %v", err)
	}

	// To get total count, need the parsed query without the paging.
	countQuery := parsedQuery.CloneVT()

	// Fill in pagination.
	paginated.FillPaginationV2(parsedQuery, query.GetPagination(), maxPaginationLimit)

	scanConfigs, err := s.scanConfigDS.GetScanConfigurations(ctx, parsedQuery)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve scan configurations for query %v", query)
	}

	scanStatuses, err := convertStorageScanConfigToV2ScanStatuses(ctx, scanConfigs, s.scanConfigDS, s.complianceScanSettingBindingsDS, s.suiteDS, s.notifierDS)
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, "failed to convert compliance scan configurations.")
	}

	scanConfigCount, err := s.scanConfigDS.CountScanConfigurations(ctx, countQuery)
	if err != nil {
		return nil, errors.Wrap(errox.NotFound, err.Error())
	}

	return &v2.ListComplianceScanConfigurationsResponse{
		Configurations: scanStatuses,
		TotalCount:     int32(scanConfigCount),
	}, nil
}

func (s *serviceImpl) GetComplianceScanConfiguration(ctx context.Context, req *v2.ResourceByID) (*v2.ComplianceScanConfigurationStatus, error) {
	if req.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Scan configuration name is required for retrieval")
	}

	scanConfig, found, err := s.scanConfigDS.GetScanConfiguration(ctx, req.GetId())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve compliance scan configuration with id %q.", req.GetId())
	}
	if !found {
		return nil, errors.Errorf("failed to retrieve compliance scan configuration with id %q.", req.GetId())
	}

	return convertStorageScanConfigToV2ScanStatus(ctx, scanConfig, s.scanConfigDS, s.complianceScanSettingBindingsDS, s.suiteDS, s.notifierDS)
}

func (s *serviceImpl) RunComplianceScanConfiguration(ctx context.Context, request *v2.ResourceByID) (*v2.Empty, error) {
	if request.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Scan configuration ID is required to rerun a scan")
	}

	err := s.manager.ProcessRescanRequest(ctx, request.GetId())
	return &v2.Empty{}, err
}

func (s *serviceImpl) RunReport(ctx context.Context, request *v2.ComplianceRunReportRequest) (*v2.ComplianceRunReportResponse, error) {
	if !features.ComplianceReporting.Enabled() {
		return nil, errors.Wrap(errox.NotImplemented, "Not implemented")
	}
	requesterID := authn.IdentityFromContextOrNil(ctx)
	if requesterID == nil {
		return nil, errors.New("Could not determine user identity from provided context")
	}
	if request.GetScanConfigId() == "" && features.ScanScheduleReportJobs.Enabled() {
		return nil, errors.Wrap(errox.InvalidArgs, "Scan configuration ID is required to run an a report")
	}

	scanConfig, found, err := s.scanConfigDS.GetScanConfiguration(ctx, request.GetScanConfigId())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve compliance scan configuration with id %q.", request.GetScanConfigId())
	}
	if !found {
		return nil, errors.Errorf("failed to retrieve compliance scan configuration with id %q.", request.GetScanConfigId())
	}

	notificationMethod, err := convertNotificationMethodToStorage(request.GetReportNotificationMethod())
	if err != nil {
		return nil, err
	}

	err = s.reportManager.SubmitReportRequest(ctx, scanConfig, notificationMethod)
	if err != nil {
		return &v2.ComplianceRunReportResponse{
			RunState:    v2.ComplianceRunReportResponse_ERROR,
			SubmittedAt: types.TimestampNow(),
			ErrorMsg:    err.Error(),
		}, errors.Wrapf(err, "failed to submit compliance on demand report request for scan config %q", scanConfig.GetScanConfigName())
	}

	return &v2.ComplianceRunReportResponse{
		RunState:    v2.ComplianceRunReportResponse_SUBMITTED,
		SubmittedAt: types.TimestampNow(),
		ErrorMsg:    "",
	}, nil
}

func (s *serviceImpl) GetReportHistory(ctx context.Context, request *v2.ComplianceReportHistoryRequest) (*v2.ComplianceReportHistoryResponse, error) {
	if !features.ComplianceReporting.Enabled() || !features.ScanScheduleReportJobs.Enabled() {
		return nil, errors.Wrapf(errox.NotImplemented, "%s or %s are not enabled", features.ComplianceReporting.EnvVar(), features.ScanScheduleReportJobs.EnvVar())
	}
	if request == nil || request.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Empty request or id")
	}
	parsedQuery, err := search.ParseQuery(request.GetReportParamQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}
	conjunctionQuery := search.ConjunctionQuery(
		search.NewQueryBuilder().AddExactMatches(
			search.ComplianceOperatorScanConfig,
			request.GetId(),
		).ProtoQuery(), parsedQuery)

	paginated.FillPaginationV2(conjunctionQuery, request.GetReportParamQuery().GetPagination(), maxPaginationLimit)

	results, err := s.snapshotDS.SearchSnapshots(ctx, conjunctionQuery)
	if err != nil {
		return nil, err
	}
	snapshots, err := convertStorageSnapshotsToV2Snapshots(ctx, results, s.scanConfigDS, s.complianceScanSettingBindingsDS, s.suiteDS, s.notifierDS, s.blobDS)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to convert storage report snapshots to response")
	}
	res := &v2.ComplianceReportHistoryResponse{
		ComplianceReportSnapshots: snapshots,
	}
	return res, nil
}

func (s *serviceImpl) GetMyReportHistory(ctx context.Context, request *v2.ComplianceReportHistoryRequest) (*v2.ComplianceReportHistoryResponse, error) {
	if !features.ComplianceReporting.Enabled() || !features.ScanScheduleReportJobs.Enabled() {
		return nil, errors.Wrapf(errox.NotImplemented, "%s or %s are not enabled", features.ComplianceReporting.EnvVar(), features.ScanScheduleReportJobs.EnvVar())
	}

	if request == nil || request.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Empty request or id")
	}

	slimUser := authn.UserFromContext(ctx)
	if slimUser == nil {
		return nil, errors.New("Could not determine user identity from provided context")
	}

	parsedQuery, err := search.ParseQuery(request.GetReportParamQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}

	conjunctionQuery := search.ConjunctionQuery(
		search.NewQueryBuilder().
			AddExactMatches(search.ComplianceOperatorScanConfig, request.GetId()).
			AddExactMatches(search.UserID, slimUser.GetId()).
			ProtoQuery(), parsedQuery)

	paginated.FillPaginationV2(conjunctionQuery, request.GetReportParamQuery().GetPagination(), maxPaginationLimit)

	results, err := s.snapshotDS.SearchSnapshots(ctx, conjunctionQuery)
	if err != nil {
		return nil, err
	}

	snapshots, err := convertStorageSnapshotsToV2Snapshots(ctx, results, s.scanConfigDS, s.complianceScanSettingBindingsDS, s.suiteDS, s.notifierDS, s.blobDS)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to convert storage report snapshots to response")
	}

	res := &v2.ComplianceReportHistoryResponse{
		ComplianceReportSnapshots: snapshots,
	}
	return res, nil
}

func (s *serviceImpl) DeleteReport(ctx context.Context, req *v2.ResourceByID) (*v2.Empty, error) {
	if !features.ComplianceReporting.Enabled() || !features.ScanScheduleReportJobs.Enabled() {
		return nil, errors.Wrapf(errox.NotImplemented, "%s or %s are not enabled", features.ComplianceReporting.EnvVar(), features.ScanScheduleReportJobs.EnvVar())
	}

	if req.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Report Snapshot ID is required for deletion")
	}

	slimUser := authn.UserFromContext(ctx)
	if slimUser == nil {
		return nil, errors.New("Could not determine user identity from provided context")
	}

	snapshot, found, err := s.snapshotDS.GetSnapshot(ctx, req.GetId())
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to retrieve Report Snapshot %s", req.GetId())
	}
	if !found {
		return nil, errors.Wrapf(errox.NotFound, "Unable to find the Report Snapshots %s", req.GetId())
	}

	if slimUser.GetId() != snapshot.GetUser().GetId() {
		return nil, errors.Errorf("The user %s cannot delete the report %s", slimUser.GetId(), snapshot.GetReportId())
	}

	status := snapshot.GetReportStatus()
	if status.GetReportNotificationMethod() != storage.ComplianceOperatorReportStatus_DOWNLOAD {
		return nil, errors.Wrapf(errox.InvalidArgs, "The Report %s is not downloadable and cannot be deleted", req.GetId())
	}
	switch status.GetRunState() {
	case storage.ComplianceOperatorReportStatus_FAILURE:
		return nil, errors.Wrapf(errox.InvalidArgs, "The Report Snapshot %s has failed and no downloadable report was generated", req.GetId())
	case storage.ComplianceOperatorReportStatus_WAITING, storage.ComplianceOperatorReportStatus_PREPARING:
		return nil, errors.Wrapf(errox.InvalidArgs, "The Report Snapshot %s is still running", req.GetId())
	}

	blobName := common.GetComplianceReportBlobPath(snapshot.GetScanConfigurationId(), req.GetId())

	ctx = sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Administration)),
	)
	if err = s.blobDS.Delete(ctx, blobName); err != nil {
		log.Errorf("Unable to delete the downloadable report: %v", err)
		return nil, errors.Wrap(errox.InvariantViolation, "Unable to delete the downloadable report")
	}

	return &v2.Empty{}, nil
}

func (s *serviceImpl) ListComplianceScanConfigProfiles(ctx context.Context, query *v2.RawQuery) (*v2.ListComplianceScanConfigsProfileResponse, error) {
	// Fill in Query.
	parsedQuery, err := search.ParseQuery(query.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to parse query %v", err)
	}

	// To get total count, need the parsed query without the paging.
	countQuery := parsedQuery.CloneVT()

	// Fill in pagination.
	paginated.FillPaginationV2(parsedQuery, query.GetPagination(), maxPaginationLimit)

	profiles, profileCount, err := s.getProfiles(ctx, parsedQuery, countQuery)
	if err != nil {
		return nil, err
	}

	return &v2.ListComplianceScanConfigsProfileResponse{
		Profiles:   profiles,
		TotalCount: int32(profileCount),
	}, nil
}

func (s *serviceImpl) ListComplianceScanConfigClusterProfiles(ctx context.Context, request *v2.ComplianceConfigClusterProfileRequest) (*v2.ListComplianceScanConfigsClusterProfileResponse, error) {
	if request.GetClusterId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "cluster is required")
	}

	clusterName, found, err := s.clusterDS.GetClusterName(ctx, request.GetClusterId())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Error retrieving cluster %q:%v", request.GetClusterId(), err)
	}
	if !found {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to find cluster %q", request.GetClusterId())
	}

	// Fill in Query.
	parsedQuery, err := search.ParseQuery(request.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to parse query %v", err)
	}

	// Add the cluster ids as an exact match
	parsedQuery = search.ConjunctionQuery(
		search.NewQueryBuilder().AddSelectFields().AddExactMatches(search.ClusterID, request.GetClusterId()).ProtoQuery(),
		parsedQuery,
	)

	// To get total count, need the parsed query without the paging.
	countQuery := parsedQuery.CloneVT()

	// Fill in pagination.
	paginated.FillPaginationV2(parsedQuery, request.GetQuery().GetPagination(), maxPaginationLimit)

	profiles, profileCount, err := s.getProfiles(ctx, parsedQuery, countQuery)
	if err != nil {
		return nil, err
	}

	return &v2.ListComplianceScanConfigsClusterProfileResponse{
		ClusterId:   request.GetClusterId(),
		ClusterName: clusterName,
		Profiles:    profiles,
		TotalCount:  int32(profileCount),
	}, nil
}

func validateScanConfiguration(req *v2.ComplianceScanConfiguration) error {
	if len(req.GetClusters()) == 0 {
		return errors.Wrap(errox.InvalidArgs, "At least one cluster is required for a scan configuration")
	}

	if req.GetScanConfig() == nil {
		return errors.Wrap(errox.InvalidArgs, "The scan configuration is nil.")
	}

	if len(req.GetScanConfig().GetProfiles()) == 0 {
		return errors.Wrap(errox.InvalidArgs, "At least one profile is required for a scan configuration")
	}

	return nil
}

func (s *serviceImpl) getBenchmarks(ctx context.Context, profileNames []string) (map[string][]*storage.ComplianceOperatorBenchmarkV2, error) {
	// Get the benchmarks
	benchmarkMap := make(map[string][]*storage.ComplianceOperatorBenchmarkV2, len(profileNames))
	for _, profileName := range profileNames {
		if _, found := benchmarkMap[profileName]; !found {
			benchmarks, err := s.benchmarkDS.GetBenchmarksByProfileName(ctx, profileName)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to retrieve benchmarks for profile %q.", profileName)
			}
			benchmarkMap[profileName] = benchmarks
		}
	}

	return benchmarkMap, nil
}

func (s *serviceImpl) getProfiles(ctx context.Context, query *v1.Query, countQuery *v1.Query) ([]*v2.ComplianceProfileSummary, int, error) {
	profileNames, err := s.scanConfigDS.GetProfilesNames(ctx, query)
	if err != nil {
		return nil, 0, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve scan configurations for query %v", query)
	}

	// Build query to get the filtered list by profile names
	profileQuery := search.NewQueryBuilder().AddSelectFields().AddExactMatches(search.ComplianceOperatorProfileName, profileNames...).ProtoQuery()
	profiles, err := s.profileDS.SearchProfiles(ctx, profileQuery)
	if err != nil {
		return nil, 0, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance profiles for %v", profileQuery)
	}

	// Get the benchmarks
	benchmarkMap, err := s.getBenchmarks(ctx, profileNames)
	if err != nil {
		return nil, 0, err
	}

	profileCount, err := s.scanConfigDS.CountDistinctProfiles(ctx, countQuery)
	if err != nil {
		return nil, 0, errors.Wrap(errox.NotFound, err.Error())
	}

	return storagetov2.ComplianceProfileSummary(profiles, benchmarkMap), profileCount, nil
}
