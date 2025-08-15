//go:build sql_integration

package m211tom212

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/m_211_to_m_212_admission_control_config/schema"
	"github.com/stackrox/rox/migrator/migrations/m_211_to_m_212_admission_control_config/test"
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

	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), schema.CreateTableClustersStmt)
}

func (s *migrationTestSuite) TestMigration() {
	clusters := []*storage.Cluster{
		s.getTestCluster("helm-cluster", true),
		s.getTestCluster("manifest-install-cluster", false),
	}

	store := test.New(s.db)
	require.NoError(s.T(), store.UpsertMany(s.ctx, clusters))

	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}

	s.Require().NoError(migration.Run(dbs))

	result, err := store.GetByQuery(s.ctx, search.EmptyQuery())
	assert.NoError(s.T(), err)
	assert.ElementsMatch(s.T(), collectIDs(clusters...), collectIDs(result...))

	result, err = store.GetByQuery(s.ctx,
		search.NewQueryBuilder().AddExactMatches(search.Cluster, clusters[0].Name).ProtoQuery())
	assert.NoError(s.T(), err)

	assert.EqualValues(s.T(), result[0].HelmConfig.DynamicConfig.AdmissionControllerConfig.ScanInline, false)
	assert.EqualValues(s.T(), result[0].HelmConfig.DynamicConfig.AdmissionControllerConfig.Enabled, true)
	assert.EqualValues(s.T(), result[0].HelmConfig.DynamicConfig.AdmissionControllerConfig.EnforceOnUpdates, false)

	result, err = store.GetByQuery(s.ctx,
		search.NewQueryBuilder().AddExactMatches(search.Cluster, clusters[1].Name).ProtoQuery())
	assert.NoError(s.T(), err)

	assert.EqualValues(s.T(), result[0].DynamicConfig.AdmissionControllerConfig.ScanInline, true)
	assert.EqualValues(s.T(), result[0].DynamicConfig.AdmissionControllerConfig.Enabled, true)
	assert.EqualValues(s.T(), result[0].DynamicConfig.AdmissionControllerConfig.EnforceOnUpdates, true)

}

func collectIDs(objs ...*storage.Cluster) []string {
	var ids []string
	for _, obj := range objs {
		ids = append(ids, obj.GetId())
	}
	return ids
}

func (s *migrationTestSuite) getTestCluster(name string, helmManaged bool) *storage.Cluster {
	c := &storage.Cluster{
		Id:        uuid.NewV4().String(),
		Name:      name,
		Type:      storage.ClusterType_OPENSHIFT_CLUSTER,
		MainImage: "quay.io/stackrox-io/main",
	}
	acc := &storage.AdmissionControllerConfig{
		Enabled:          true,
		TimeoutSeconds:   0,
		ScanInline:       false,
		DisableBypass:    false,
		EnforceOnUpdates: false,
	}

	if helmManaged {
		c.HelmConfig = &storage.CompleteClusterConfig{
			DynamicConfig: &storage.DynamicClusterConfig{
				AdmissionControllerConfig: acc,
			},
		}
	} else {
		c.DynamicConfig = &storage.DynamicClusterConfig{
			AdmissionControllerConfig: acc,
		}
	}
	return c
}
