//go:build sql_integration

package m204tom205

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	oldSchema "github.com/stackrox/rox/migrator/migrations/m_204_to_m_205_clusters_platform_type_and_k8_version/schema/old"
	previousStore "github.com/stackrox/rox/migrator/migrations/m_204_to_m_205_clusters_platform_type_and_k8_version/store/previous"
	updatedStore "github.com/stackrox/rox/migrator/migrations/m_204_to_m_205_clusters_platform_type_and_k8_version/store/updated"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type migrationTestSuite struct {
	suite.Suite

	db  *pghelper.TestPostgres
	ctx context.Context
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(migrationTestSuite))
}

func (s *migrationTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
	s.db = pghelper.ForT(s.T(), false)

	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), oldSchema.CreateTableClustersStmt)
}

func (s *migrationTestSuite) TestMigration() {
	clusters := []*storage.Cluster{
		s.getTestCluster("generic-1", storage.ClusterType_GENERIC_CLUSTER, "9.0"),
		s.getTestCluster("generic-2", storage.ClusterType_GENERIC_CLUSTER, "9.0"),
		s.getTestCluster("kubernetes-1", storage.ClusterType_KUBERNETES_CLUSTER, "9.0"),
		s.getTestCluster("kubernetes-2", storage.ClusterType_KUBERNETES_CLUSTER, "9.5"),
		s.getTestCluster("openshift-1", storage.ClusterType_OPENSHIFT_CLUSTER, "9.5"),
		s.getTestCluster("openshift4-1", storage.ClusterType_OPENSHIFT4_CLUSTER, "9.5"),
	}

	prevStore := previousStore.New(s.db)
	require.NoError(s.T(), prevStore.UpsertMany(s.ctx, clusters))

	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}

	s.Require().NoError(migration.Run(dbs))

	newStore := updatedStore.New(s.db)
	result, err := newStore.GetByQuery(s.ctx, search.EmptyQuery())
	assert.NoError(s.T(), err)
	assert.ElementsMatch(s.T(), collectIDs(clusters...), collectIDs(result...))

	result, err = newStore.GetByQuery(s.ctx,
		search.NewQueryBuilder().AddExactMatches(search.ClusterPlatformType, storage.ClusterType_KUBERNETES_CLUSTER.String()).ProtoQuery())
	assert.NoError(s.T(), err)
	assert.ElementsMatch(s.T(), collectIDs(clusters[2], clusters[3]), collectIDs(result...))

	result, err = newStore.GetByQuery(s.ctx,
		search.NewQueryBuilder().AddExactMatches(search.ClusterPlatformType, storage.ClusterType_OPENSHIFT4_CLUSTER.String()).ProtoQuery())
	assert.NoError(s.T(), err)
	assert.ElementsMatch(s.T(), collectIDs(clusters[5]), collectIDs(result...))

	result, err = newStore.GetByQuery(s.ctx,
		search.NewQueryBuilder().AddExactMatches(search.ClusterKubernetesVersion, "9.5").ProtoQuery())
	assert.NoError(s.T(), err)
	assert.ElementsMatch(s.T(), collectIDs(clusters[3], clusters[4], clusters[5]), collectIDs(result...))
}

func collectIDs(objs ...*storage.Cluster) []string {
	var ids []string
	for _, obj := range objs {
		ids = append(ids, obj.GetId())
	}
	return ids
}

func (s *migrationTestSuite) getTestCluster(name string, platformType storage.ClusterType, k8sVersion string) *storage.Cluster {
	return &storage.Cluster{
		Id:        uuid.NewV4().String(),
		Name:      name,
		Type:      platformType,
		Labels:    map[string]string{"key": "val"},
		MainImage: "quay.io/stackrox-io/main",
		Status: &storage.ClusterStatus{
			OrchestratorMetadata: &storage.OrchestratorMetadata{
				Version: k8sVersion,
			},
		},
	}
}
