package datastore

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	statusStore "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/scanconfigstatus/store/postgres"
	"github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/concurrency"
	pgPkg "github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	complianceSAC = sac.ForResource(resources.Compliance)
)

type datastoreImpl struct {
	db            pgPkg.DB
	storage       postgres.Store
	statusStorage statusStore.Store
	keyedMutex    *concurrency.KeyedMutex
}

// GetScanConfiguration retrieves the scan configuration specified by id
func (ds *datastoreImpl) GetScanConfiguration(ctx context.Context, id string) (*storage.ComplianceOperatorScanConfigurationV2, bool, error) {
	scanConfig, found, err := ds.storage.Get(ctx, id)

	// We must ensure the user has access to all the clusters in a config.  The SAC filter will return the row
	// if the user has access to any cluster
	if !complianceSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS).AllAllowed(getScopeKeys(scanConfig.GetClusters())) {
		return nil, false, nil
	}

	return scanConfig, found, err
}

// GetScanConfigurationByName retrieves the scan configuration specified by name
func (ds *datastoreImpl) GetScanConfigurationByName(ctx context.Context, scanName string) (*storage.ComplianceOperatorScanConfigurationV2, error) {
	scanConfigs, err := ds.storage.GetByQuery(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorScanConfigName, scanName).ProtoQuery())
	if err != nil {
		return nil, err
	}
	if len(scanConfigs) == 0 {
		return nil, nil
	}

	if len(scanConfigs) > 1 {
		return nil, errors.Errorf("unable to retrieve distinct scan configuration named %q", scanName)
	}

	return scanConfigs[0], nil
}

// ScanConfigurationProfileExists takes all the profiles being referenced by the scan configuration and checks if any cluster in the configuration is using it in any existing scan configurations.
func (ds *datastoreImpl) ScanConfigurationProfileExists(ctx context.Context, id string, profiles []string, clusters []string) error {
	for i := 0; i < len(profiles); i++ {
		for j := i + 1; j < len(profiles); j++ {
			if strings.EqualFold(profiles[i], profiles[j]) {
				return errors.Errorf("the scan configuration contains duplicate profiles.  Profile %q and profile %q", profiles[i], profiles[j])
			}
		}
	}

	// Retrieve all scan configurations for the specified clusters.
	scanConfigs, err := ds.storage.GetByQuery(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, clusters...).ProtoQuery())
	if err != nil {
		return err
	}

	// Create a map for quick lookup of profiles.
	profileMap := make(map[string]set.StringSet)

	for _, scanConfig := range scanConfigs {
		if scanConfig.GetId() == id {
			continue
		}
		for _, profile := range scanConfig.GetProfiles() {
			configSet, found := profileMap[profile.GetProfileName()]
			if !found {
				profileMap[profile.GetProfileName()] = set.NewStringSet()
			}
			configSet.Add(scanConfig.GetScanConfigName())
			profileMap[profile.GetProfileName()] = configSet
		}
	}

	// Check if any of the profiles are being used by any of the existing scan configurations.
	for _, profile := range profiles {
		for profileName, configs := range profileMap {
			if strings.EqualFold(profile, profileName) {
				return errors.Errorf("a cluster in scan configurations %v already uses profile %q", configs.AsSlice(), profileName)
			}
		}
	}

	return nil
}

// GetScanConfigurations retrieves the scan configurations specified by query
func (ds *datastoreImpl) GetScanConfigurations(ctx context.Context, query *v1.Query) ([]*storage.ComplianceOperatorScanConfigurationV2, error) {
	scanConfigs, err := ds.storage.GetByQuery(ctx, query)

	// SAC will return a config if a user has permissions to ANY of the clusters.  For tech preview, and
	// in the interest of ensuring we don't leak clusters, if a user does not have access to one or more
	// of the clusters returned by the query, we will return nothing.  An all or nothing approach in the
	// interest of not leaking data.
	for _, scanConfig := range scanConfigs {
		// We must ensure the user has access to all the clusters in a config.  The SAC filter will return the row
		// if the user has access to any cluster
		if !complianceSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS).AllAllowed(getScopeKeys(scanConfig.GetClusters())) {
			return nil, nil
		}
	}

	return scanConfigs, err
}

// UpsertScanConfiguration adds or updates the scan configuration
func (ds *datastoreImpl) UpsertScanConfiguration(ctx context.Context, scanConfig *storage.ComplianceOperatorScanConfigurationV2) error {
	// SAC for an upsert requires access to all clusters present in the conifg.  This is handled
	// in the store so a SAC check is not needed here.

	ds.keyedMutex.Lock(scanConfig.GetId())
	defer ds.keyedMutex.Unlock(scanConfig.GetId())

	// Update the last updated time
	return ds.upsertNoLockScanConfiguration(ctx, scanConfig)
}

// upsertNoLockScanConfiguration upserts scan config like UpsertScanConfiguration but does not create a lock
func (ds *datastoreImpl) upsertNoLockScanConfiguration(ctx context.Context, scanConfig *storage.ComplianceOperatorScanConfigurationV2) error {
	scanConfig.LastUpdatedTime = protocompat.TimestampNow()
	return ds.storage.Upsert(ctx, scanConfig)
}

// DeleteScanConfiguration deletes the scan configuration specified by id
func (ds *datastoreImpl) DeleteScanConfiguration(ctx context.Context, id string) (string, error) {
	// Need to verify that write to all clusters used in this configuration is allowed.
	elevatedSACReadCtx := sac.WithAllAccess(context.Background())

	// Use elevated privileges to get all clusters associated with this configuration.
	scanConfig, found, err := ds.GetScanConfiguration(elevatedSACReadCtx, id)
	if err != nil {
		return "", errors.Wrapf(err, "Unable to find scan configuration id %q", id)
	}
	if !found {
		return "", errors.Errorf("Scan configuration id %q not found", id)
	}
	scanConfigName := scanConfig.GetScanConfigName()

	if !complianceSAC.ScopeChecker(ctx, storage.Access_READ_WRITE_ACCESS).AllAllowed(getScopeKeys(scanConfig.GetClusters())) {
		return "", sac.ErrResourceAccessDenied
	}

	ds.keyedMutex.Lock(id)
	defer ds.keyedMutex.Unlock(id)

	// remove scan data from scan status table first
	_, err = ds.statusStorage.DeleteByQuery(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorScanConfig, id).ProtoQuery())
	if err != nil {
		return "", errors.Wrapf(err, "Unable to delete scan status for scan configuration id %q", id)
	}

	err = ds.storage.Delete(ctx, id)
	if err != nil {
		return "", errors.Wrapf(err, "Unable to delete scan configuration id %q", id)
	}

	return scanConfigName, nil
}

// UpdateClusterStatus updates the scan configuration with the cluster status
func (ds *datastoreImpl) UpdateClusterStatus(ctx context.Context, scanConfigID string, clusterID string, clusterStatus string, clusterName string) error {
	if !complianceSAC.ScopeChecker(ctx, storage.Access_READ_WRITE_ACCESS).IsAllowed(sac.ClusterScopeKey(clusterID)) {
		return sac.ErrResourceAccessDenied
	}

	ds.keyedMutex.Lock(scanConfigID)
	defer ds.keyedMutex.Unlock(scanConfigID)

	// Ensure the scan configuration exists
	_, found, err := ds.GetScanConfiguration(ctx, scanConfigID)
	if err != nil {
		return errors.Wrapf(err, "Unable to retrieve scan configuration id %q", scanConfigID)
	}
	if !found {
		return errors.Errorf("Unable to find scan configuration id %q", scanConfigID)
	}

	// Need to build a deterministic ID from clusterID and scanID to ensure we always have the latest status
	clusterUUID, err := uuid.FromString(clusterID)
	if err != nil {
		return errors.Wrapf(err, "Unable to build scan configuration status id based off %q", scanConfigID)
	}
	statusKey := uuid.NewV5(clusterUUID, scanConfigID).String()

	clusterScanStatus := &storage.ComplianceOperatorClusterScanConfigStatus{
		Id:           statusKey,
		ClusterId:    clusterID,
		ClusterName:  clusterName,
		ScanConfigId: scanConfigID,
		Errors:       []string{clusterStatus},
	}

	return ds.statusStorage.Upsert(ctx, clusterScanStatus)
}

// RemoveClusterStatus removes the scan configuration status for the given cluster
func (ds *datastoreImpl) RemoveClusterStatus(ctx context.Context, scanConfigID string, clusterID string) error {
	_, err := ds.statusStorage.DeleteByQuery(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorScanConfig, scanConfigID).
		AddExactMatches(search.ClusterID, clusterID).
		ProtoQuery())

	return err
}

// GetScanConfigClusterStatus retrieves the scan configurations status per cluster specified by scan id
func (ds *datastoreImpl) GetScanConfigClusterStatus(ctx context.Context, scanConfigID string) ([]*storage.ComplianceOperatorClusterScanConfigStatus, error) {
	return ds.statusStorage.GetByQuery(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorScanConfig, scanConfigID).ProtoQuery())
}

func (ds *datastoreImpl) CountScanConfigurations(ctx context.Context, q *v1.Query) (int, error) {
	// Need to account for cluster SAC, so first get the configs with the SAC filters applied
	scanConfigs, err := ds.GetScanConfigurations(ctx, q)
	return len(scanConfigs), err
}

func getScopeKeys(scanClusters []*storage.ComplianceOperatorScanConfigurationV2_Cluster) [][]sac.ScopeKey {
	clusterScopeKeys := make([][]sac.ScopeKey, 0, len(scanClusters))
	for _, scanCluster := range scanClusters {
		clusterScopeKeys = append(clusterScopeKeys, []sac.ScopeKey{sac.ClusterScopeKey(scanCluster.GetClusterId())})
	}

	return clusterScopeKeys
}

func (ds *datastoreImpl) deleteClusterFromScanConfigWithLock(ctx context.Context, clusterID string, scanConfig *storage.ComplianceOperatorScanConfigurationV2) error {
	ds.keyedMutex.Lock(scanConfig.GetId())
	defer ds.keyedMutex.Unlock(scanConfig.GetId())

	clusters := scanConfig.GetClusters()
	filterFunction := func(cluster *storage.ComplianceOperatorScanConfigurationV2_Cluster) bool {
		return cluster.GetClusterId() != clusterID
	}
	newClusters := sliceutils.Filter(clusters, filterFunction)
	scanConfig.Clusters = newClusters

	err := ds.upsertNoLockScanConfiguration(ctx, scanConfig)
	if err != nil {
		return err
	}

	// Remove the status for the cluster as well.
	return ds.RemoveClusterStatus(ctx, scanConfig.GetId(), clusterID)
}

func (ds *datastoreImpl) RemoveClusterFromScanConfig(ctx context.Context, clusterID string) error {
	q := search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, clusterID).ProtoQuery()
	scans, err := ds.GetScanConfigurations(ctx, q)
	if err != nil {
		return err
	}

	for _, scan := range scans {
		err = ds.deleteClusterFromScanConfigWithLock(ctx, clusterID, scan)
		if err != nil {
			return err
		}
	}
	return nil
}

type distinctProfileName struct {
	ProfileName string `db:"compliance_config_profile_name"`
}

// GetProfilesNames gets the list of distinct profile names for the query
func (ds *datastoreImpl) GetProfilesNames(ctx context.Context, q *v1.Query) ([]string, error) {
	var err error
	q, err = withSACFilter(ctx, resources.Compliance, q)
	if err != nil {
		return nil, err
	}

	clonedQuery := q.CloneVT()

	// Build the select and group by on distinct profile name
	clonedQuery.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.ComplianceOperatorConfigProfileName).Distinct().Proto(),
	}
	clonedQuery.GroupBy = &v1.QueryGroupBy{
		Fields: []string{
			search.ComplianceOperatorConfigProfileName.String(),
		},
	}

	clonedQuery.Pagination = q.GetPagination()

	var results []*distinctProfileName
	results, err = pgSearch.RunSelectRequestForSchema[distinctProfileName](ctx, ds.db, schema.ComplianceOperatorScanConfigurationV2Schema, clonedQuery)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}
	profileNames := make([]string, 0, len(results))
	for _, result := range results {
		profileNames = append(profileNames, result.ProfileName)
	}

	return profileNames, err
}

type distinctProfileCount struct {
	TotalCount int    `db:"compliance_config_profile_name_count"`
	Name       string `db:"compliance_config_profile_name"`
}

// CountDistinctProfiles returns count of distinct profiles matching query
func (ds *datastoreImpl) CountDistinctProfiles(ctx context.Context, q *v1.Query) (int, error) {
	var err error
	q, err = withSACFilter(ctx, resources.Compliance, q)
	if err != nil {
		return 0, err
	}

	query := q.CloneVT()

	query.GroupBy = &v1.QueryGroupBy{
		Fields: []string{
			search.ComplianceOperatorConfigProfileName.String(),
		},
	}

	var results []*distinctProfileCount
	results, err = pgSearch.RunSelectRequestForSchema[distinctProfileCount](ctx, ds.db, schema.ComplianceOperatorScanConfigurationV2Schema, withCountQuery(query, search.ComplianceOperatorConfigProfileName))
	if err != nil {
		return 0, err
	}
	return len(results), nil
}

func withCountQuery(query *v1.Query, field search.FieldLabel) *v1.Query {
	cloned := query.CloneVT()
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(field).AggrFunc(aggregatefunc.Count).Proto(),
	}
	return cloned
}

func withSACFilter(ctx context.Context, targetResource permissions.ResourceMetadata, query *v1.Query) (*v1.Query, error) {
	sacQueryFilter, err := pgSearch.GetReadSACQuery(ctx, targetResource)
	if err != nil {
		return nil, err
	}
	return search.FilterQueryByQuery(query, sacQueryFilter), nil
}
