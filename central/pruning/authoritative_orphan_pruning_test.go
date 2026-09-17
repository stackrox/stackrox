//go:build sql_integration

package pruning

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	alertDatastore "github.com/stackrox/rox/central/alert/datastore"
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	clusterPostgres "github.com/stackrox/rox/central/cluster/store/cluster/postgres"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	networkFlowDatastore "github.com/stackrox/rox/central/networkgraph/flow/datastore"
	nodeDatastore "github.com/stackrox/rox/central/node/datastore"
	podDatastore "github.com/stackrox/rox/central/pod/datastore"
	processBaselineDatastore "github.com/stackrox/rox/central/processbaseline/datastore"
	baselinePostgres "github.com/stackrox/rox/central/processbaseline/store/postgres"
	baselineResultsDatastore "github.com/stackrox/rox/central/processbaselineresults/datastore"
	processDatastore "github.com/stackrox/rox/central/processindicator/datastore"
	plopDatastore "github.com/stackrox/rox/central/processlisteningonport/datastore"
	roleDatastore "github.com/stackrox/rox/central/rbac/k8srole/datastore"
	bindingDatastore "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	serviceAccountDatastore "github.com/stackrox/rox/central/serviceaccount/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newOrphanPruningCollector(t *testing.T, db postgres.DB) *garbageCollectorImpl {
	t.Helper()
	clusters, err := clusterDatastore.GetTestPostgresDataStore(t, db)
	require.NoError(t, err)
	deployments, err := deploymentDatastore.GetTestPostgresDataStore(t, db)
	require.NoError(t, err)
	flows, err := networkFlowDatastore.GetTestPostgresClusterDataStore(t, db)
	require.NoError(t, err)
	return &garbageCollectorImpl{
		postgres:        db,
		clusters:        clusters,
		deployments:     deployments,
		networkflows:    flows,
		alerts:          alertDatastore.GetTestPostgresDataStore(t, db),
		nodes:           nodeDatastore.GetTestPostgresDataStore(t, db),
		pods:            podDatastore.GetTestPostgresDataStore(t, db),
		processes:       processDatastore.GetTestPostgresDataStore(t, db),
		plops:           plopDatastore.GetTestPostgresDataStore(t, db),
		processbaseline: processBaselineDatastore.GetTestPostgresDataStore(t, db),
		serviceAccts:    serviceAccountDatastore.GetTestPostgresDataStore(t, db),
		k8sRoles:        roleDatastore.GetTestPostgresDataStore(t, db),
		k8sRoleBindings: bindingDatastore.GetTestPostgresDataStore(t, db),
	}
}

// Queries execute on a real failed PostgreSQL transaction, while the datastores
// keep healthy connections. Falling back to datastore searches would delete rows.
type failedPruningDB struct {
	postgres.DB
	tx *postgres.Tx
}

func (db failedPruningDB) Query(ctx context.Context, query string, args ...interface{}) (*postgres.Rows, error) {
	return db.tx.Query(ctx, query, args...)
}

func (db failedPruningDB) Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	return db.tx.Exec(ctx, query, args...)
}

func assertPruningRows(t *testing.T, db postgres.DB, table string, expected []string) {
	t.Helper()
	rows, err := db.Query(context.Background(), "SELECT id FROM "+table)
	require.NoError(t, err)
	ids, err := pgutils.ScanStrings(rows)
	require.NoError(t, err)
	assert.ElementsMatch(t, expected, ids, table)
}

func TestRemoveOrphanedResourcesAuthoritative(t *testing.T) {
	for name, tc := range map[string]struct {
		parents        int
		cancelled      bool
		failed         bool
		workerBaseline bool
	}{
		"zero parents":                           {},
		"one parent":                             {parents: 1},
		"multiple parents":                       {parents: 2},
		"cancelled queries preserve all rows":    {parents: 1, cancelled: true},
		"failed queries preserve all rows":       {parents: 1, failed: true},
		"worker baseline store predates inserts": {parents: 1, workerBaseline: true},
	} {
		t.Run(name, func(t *testing.T) {
			db := pgtest.ForT(t).DB
			ctx := sac.WithAllAccess(context.Background())
			workerClusters := clusterPostgres.New(db)
			gc := newOrphanPruningCollector(t, db)
			centralClusters := clusterPostgres.New(db)
			centralDeployments, err := deploymentDatastore.GetTestPostgresDataStore(t, db)
			require.NoError(t, err)
			clusters := []string{fixtureconsts.Cluster1, fixtureconsts.Cluster2, fixtureconsts.Cluster3}
			deployments := []string{fixtureconsts.Deployment1, fixtureconsts.Deployment2, fixtureconsts.Deployment3}
			for i := 0; i < tc.parents; i++ {
				require.NoError(t, centralClusters.Upsert(ctx, &storage.Cluster{Id: clusters[i], Name: fmt.Sprintf("live-%d", i)}))
				require.NoError(t, centralDeployments.UpsertDeployment(ctx, &storage.Deployment{Id: deployments[i], ClusterId: clusters[i]}))
			}
			ids, err := workerClusters.GetIDs(ctx)
			require.NoError(t, err)
			require.Empty(t, ids, "independent cluster startup snapshot remains stale")
			cachedClusters, err := gc.clusters.GetClusters(ctx)
			require.NoError(t, err)
			require.Empty(t, cachedClusters)
			ids, err = gc.deployments.GetDeploymentIDs(ctx)
			require.NoError(t, err)
			require.Empty(t, ids, "independent deployment startup snapshot remains stale")

			serviceAccounts := serviceAccountDatastore.GetTestPostgresDataStore(t, db)
			roles := roleDatastore.GetTestPostgresDataStore(t, db)
			bindings := bindingDatastore.GetTestPostgresDataStore(t, db)
			baselines := baselinePostgres.New(db)
			results := baselineResultsDatastore.GetTestPostgresDataStore(t, db)
			gc.processbaseline = processBaselineDatastore.New(baselines, results, gc.processes)
			if tc.workerBaseline {
				gc.processbaseline = processBaselineDatastore.New(baselinePostgres.New(pgSearch.WithoutStoreCache(db)), results, gc.processes)
			}
			var expectedRBAC, expectedBaselines, expectedResults []string
			for i, clusterID := range clusters {
				id := deployments[i]
				require.NoError(t, serviceAccounts.UpsertServiceAccount(ctx, &storage.ServiceAccount{Id: id, ClusterId: clusterID}))
				require.NoError(t, roles.UpsertRole(ctx, &storage.K8SRole{Id: id, ClusterId: clusterID}))
				require.NoError(t, bindings.UpsertRoleBinding(ctx, &storage.K8SRoleBinding{
					Id: id, ClusterId: clusterID, Subjects: []*storage.Subject{{Name: "subject"}},
				}))
				key := &storage.ProcessBaselineKey{ClusterId: clusterID, Namespace: "ns", DeploymentId: id, ContainerName: "old"}
				baselineID := fmt.Sprintf("DC:%s:ns:%s:old", clusterID, id)
				require.NoError(t, baselines.Upsert(ctx, &storage.ProcessBaseline{
					Id: baselineID, Key: key, Created: protoconv.ConvertTimeToTimestamp(time.Now().Add(-2 * orphanWindow)),
				}))
				require.NoError(t, results.UpsertBaselineResults(ctx, &storage.ProcessBaselineResults{DeploymentId: id, ClusterId: clusterID, Namespace: "ns"}))
				if i < tc.parents || tc.cancelled || tc.failed {
					expectedRBAC = append(expectedRBAC, id)
					expectedBaselines = append(expectedBaselines, baselineID)
					expectedResults = append(expectedResults, id)
				}
			}
			// A recent baseline protects its results even when an older sibling is pruned.
			recentID := fmt.Sprintf("DC:%s:ns:%s:recent", clusters[2], deployments[2])
			require.NoError(t, baselines.Upsert(ctx, &storage.ProcessBaseline{
				Id: recentID, Key: &storage.ProcessBaselineKey{ClusterId: clusters[2], Namespace: "ns", DeploymentId: deployments[2], ContainerName: "recent"},
				Created: protocompat.TimestampNow(),
			}))
			expectedBaselines = append(expectedBaselines, recentID)
			withoutCreatedID := fmt.Sprintf("DC:%s:ns:%s:without-created", clusters[2], deployments[2])
			require.NoError(t, baselines.Upsert(ctx, &storage.ProcessBaseline{
				Id: withoutCreatedID, Key: &storage.ProcessBaselineKey{ClusterId: clusters[2], Namespace: "ns", DeploymentId: deployments[2], ContainerName: "without-created"},
			}))
			expectedBaselines = append(expectedBaselines, withoutCreatedID)
			if !tc.cancelled && !tc.failed {
				expectedResults = append(expectedResults, deployments[2])
			}

			if tc.cancelled {
				originalCtx := pruningCtx
				cancelledCtx, cancel := context.WithCancel(ctx)
				cancel()
				pruningCtx = cancelledCtx
				t.Cleanup(func() { pruningCtx = originalCtx })
			}
			if tc.failed {
				t.Setenv(env.PostgresDisableQueryRetries.EnvVar(), "true")
				failurePool, err := postgres.New(ctx, db.Config())
				require.NoError(t, err)
				t.Cleanup(failurePool.Close)
				tx, err := failurePool.Begin(ctx)
				require.NoError(t, err)
				t.Cleanup(func() { require.NoError(t, tx.Rollback(ctx)) })
				_, err = tx.Exec(ctx, "SELECT 1 / 0")
				require.Error(t, err)
				gc.postgres = failedPruningDB{DB: db, tx: tx}
			}
			gc.removeOrphanedResources()

			for _, table := range []string{schema.ServiceAccountsTableName, schema.K8sRolesTableName, schema.RoleBindingsTableName} {
				assertPruningRows(t, db, table, expectedRBAC)
			}
			assertPruningRows(t, db, schema.ProcessBaselinesTableName, expectedBaselines)
			// Results use deploymentid as their primary key.
			rows, err := db.Query(ctx, "SELECT deploymentid FROM "+schema.ProcessBaselineResultsTableName)
			require.NoError(t, err)
			resultIDs, err := pgutils.ScanStrings(rows)
			require.NoError(t, err)
			assert.ElementsMatch(t, expectedResults, resultIDs)
			var subjectCount int
			require.NoError(t, db.QueryRow(ctx, "SELECT count(*) FROM "+schema.RoleBindingsSubjectsTableName).Scan(&subjectCount))
			assert.Equal(t, len(expectedRBAC), subjectCount, "role-binding subjects must cascade on deletion")
		})
	}
}

func TestRemoveOrphanedResourcesAfterDeploymentDeletion(t *testing.T) {
	db := pgtest.ForT(t).DB
	ctx := sac.WithAllAccess(context.Background())
	centralDeployments, err := deploymentDatastore.GetTestPostgresDataStore(t, db)
	require.NoError(t, err)
	require.NoError(t, centralDeployments.UpsertDeployment(ctx, &storage.Deployment{Id: fixtureconsts.Deployment1}))
	baselines := baselinePostgres.New(db)
	results := baselineResultsDatastore.GetTestPostgresDataStore(t, db)
	for _, containerName := range []string{"old", "older"} {
		baseline := &storage.ProcessBaseline{
			Id:      fmt.Sprintf("DC:%s:ns:%s:%s", fixtureconsts.Cluster1, fixtureconsts.Deployment1, containerName),
			Key:     &storage.ProcessBaselineKey{ClusterId: fixtureconsts.Cluster1, Namespace: "ns", DeploymentId: fixtureconsts.Deployment1, ContainerName: containerName},
			Created: protoconv.ConvertTimeToTimestamp(time.Now().Add(-2 * orphanWindow)),
		}
		require.NoError(t, baselines.Upsert(ctx, baseline))
	}
	require.NoError(t, results.UpsertBaselineResults(ctx, &storage.ProcessBaselineResults{
		DeploymentId: fixtureconsts.Deployment1, ClusterId: fixtureconsts.Cluster1, Namespace: "ns",
	}))
	gc := newOrphanPruningCollector(t, db)
	// Simulate deletion by another process without running its baseline cleanup.
	_, err = db.Exec(ctx, "DELETE FROM "+schema.DeploymentsTableName+" WHERE id = $1", fixtureconsts.Deployment1)
	require.NoError(t, err)
	ids, err := gc.deployments.GetDeploymentIDs(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{fixtureconsts.Deployment1}, ids, "worker still caches the deleted deployment")

	gc.removeOrphanedResources()
	assertPruningRows(t, db, schema.ProcessBaselinesTableName, nil)
	var resultCount int
	require.NoError(t, db.QueryRow(ctx, "SELECT count(*) FROM "+schema.ProcessBaselineResultsTableName).Scan(&resultCount))
	assert.Zero(t, resultCount, "removing the last baseline must remove its results")
}
