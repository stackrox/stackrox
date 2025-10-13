//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
)

var (
	hasComplianceCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Compliance)))
)

type clusterInfo struct {
	clusterID   string
	clusterName string
	scanRefID   string
}

func cleanupDatabase(b *testing.B, pool *pgtest.TestPostgres) {
	_, err := pool.DB.Exec(context.Background(), "truncate compliance_operator_scan_configuration_v2_profiles")
	require.NoError(b, err)
	_, err = pool.DB.Exec(context.Background(), "truncate compliance_operator_scan_configuration_v2_clusters")
	require.NoError(b, err)
	_, err = pool.DB.Exec(context.Background(), "truncate compliance_operator_cluster_scan_config_statuses")
	require.NoError(b, err)
	_, err = pool.DB.Exec(context.Background(), "delete from compliance_operator_scan_configuration_v2")
	require.NoError(b, err)

	_, err = pool.DB.Exec(context.Background(), "delete from clusters")
	require.NoError(b, err)

	_, err = pool.DB.Exec(context.Background(), "truncate compliance_operator_check_result_v2")
	require.NoError(b, err)
}

func setupTest(b *testing.B, pool *pgtest.TestPostgres, datastore DataStore, numConfigs int, numClusters int, numResults int) {
	scanConfigs := make([]string, 0, numConfigs)
	for i := 0; i < numConfigs; i++ {
		scanConfigName := fmt.Sprintf("scan-config-%d", i)
		_, err := pool.DB.Exec(context.Background(), "insert into compliance_operator_scan_configuration_v2 (id, scanconfigname) values ($1, $2)", uuid.NewV4().String(), scanConfigName)
		require.NoError(b, err)

		scanConfigs = append(scanConfigs, scanConfigName)
	}

	clusters := make([]clusterInfo, 0, numClusters)
	profiles := make(map[string]string, numClusters)
	for i := 0; i < numClusters; i++ {
		clusterID := uuid.NewV4().String()
		clusterName := fmt.Sprintf("cluster-%d", i)
		scanRefID := uuid.NewV4().String()
		_, err := pool.DB.Exec(context.Background(), "insert into clusters (id, name) values ($1, $2)", clusterID, clusterName)
		require.NoError(b, err)

		clusters = append(clusters, clusterInfo{
			clusterID:   clusterID,
			clusterName: clusterName,
			scanRefID:   scanRefID,
		})

		profileRefID := uuid.NewV4().String()
		profiles[clusterID] = profileRefID
		_, err = pool.DB.Exec(context.Background(), "insert into compliance_operator_profile_v2 (id, profileid, name, producttype, clusterid, profilerefid) values ($1, $2, $3, $4, $5, $6)", uuid.NewV4().String(), "profile-1", "ocp4-cis-node", "node", clusterID, profileRefID)
		require.NoError(b, err)

		_, err = pool.DB.Exec(context.Background(), "insert into compliance_operator_scan_v2 (id, scanconfigname, scanname, profile_profilerefid, clusterid, scanrefid) values ($1, $2, $3, $4, $5, $6)", uuid.NewV4().String(), scanConfigs[0], scanConfigs[0], profileRefID, clusterID, scanRefID)
		require.NoError(b, err)
	}

	for i := 0; i < numResults; i++ {
		resultName := fmt.Sprintf("check-result-%d", i)
		for _, cluster := range clusters {
			for _, scanConfig := range scanConfigs {
				require.NoError(b, datastore.UpsertResult(hasComplianceCtx, fixtures.GetComplianceCheckResult(resultName, cluster.clusterID, cluster.clusterName, scanConfig, scanConfig, cluster.scanRefID)))
			}
		}

	}
}

func BenchmarkComplianceCheckResultStats(b *testing.B) {
	pool := pgtest.ForT(b)
	datastore := GetTestPostgresDataStore(b, pool)

	setupTest(b, pool, datastore, 5, 10, 500)
	b.Run("ComplianceCheckResultStats_5_10_500", func(b *testing.B) {
		results, err := datastore.ComplianceCheckResultStats(hasComplianceCtx, search.EmptyQuery())
		require.NoError(b, err)
		require.NotEmpty(b, results)
	})
	b.Run("ComplianceClusterStats_5_10_500", func(b *testing.B) {
		results, err := datastore.ComplianceClusterStats(hasComplianceCtx, search.EmptyQuery())
		require.NoError(b, err)
		require.NotEmpty(b, results)
	})
	cleanupDatabase(b, pool)

	setupTest(b, pool, datastore, 5, 100, 500)
	b.Run("ComplianceCheckResultStats_5_100_500", func(b *testing.B) {
		results, err := datastore.ComplianceCheckResultStats(hasComplianceCtx, search.EmptyQuery())
		require.NoError(b, err)
		require.NotEmpty(b, results)
	})
	b.Run("ComplianceClusterStats_5_100_500", func(b *testing.B) {
		results, err := datastore.ComplianceClusterStats(hasComplianceCtx, search.EmptyQuery())
		require.NoError(b, err)
		require.NotEmpty(b, results)
	})
	cleanupDatabase(b, pool)

	setupTest(b, pool, datastore, 5, 500, 500)
	b.Run("ComplianceCheckResultStats_5_500_500", func(b *testing.B) {
		results, err := datastore.ComplianceCheckResultStats(hasComplianceCtx, search.EmptyQuery())
		require.NoError(b, err)
		require.NotEmpty(b, results)
	})
	b.Run("ComplianceClusterStats_5_500_500", func(b *testing.B) {
		results, err := datastore.ComplianceClusterStats(hasComplianceCtx, search.EmptyQuery())
		require.NoError(b, err)
		require.NotEmpty(b, results)
	})
}
