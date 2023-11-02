//go:build sql_integration

package m194tom195

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/m_194_to_m_195_vuln_request_global_scope/schema"
	"github.com/stackrox/rox/migrator/migrations/m_194_to_m_195_vuln_request_global_scope/store"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
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

	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), schema.CreateTableVulnerabilityRequestsStmt)
}

func (s *migrationTestSuite) TearDownSuite() {
	s.db.Teardown(s.T())
}

func (s *migrationTestSuite) TestMigration() {
	imageScopedReqs := []*storage.VulnerabilityRequest{
		fixtures.GetImageScopeDeferralRequest("reg-1", "remote-1", "tag-1", "cve-1"),
		fixtures.GetImageScopeDeferralRequest("reg-2", "remote-1", "tag-1", "cve-2"),
		fixtures.GetImageScopeDeferralRequest("reg-3", "remote-1", "tag-1", ""),
		fixtures.GetImageScopeDeferralRequest("reg-4", "remote-1", "", ""),
		fixtures.GetImageScopeDeferralRequest("reg-5", "", "", ""),
		fixtures.GetImageScopeDeferralRequest("reg-6", "remote-2", ".*", "cve-1"),
	}

	globalScopedReqs := []*storage.VulnerabilityRequest{
		fixtures.GetGlobalFPRequest("cve-1"),
		fixtures.GetGlobalFPRequest(""),
		fixtures.GetGlobalDeferralRequest("cve-1"),
		fixtures.GetGlobalDeferralRequest(""),
	}
	var ids []string
	for _, req := range imageScopedReqs {
		ids = append(ids, req.GetId())
	}
	for _, req := range globalScopedReqs {
		ids = append(ids, req.GetId())
	}

	vulnReqStore := store.New(s.db)
	require.NoError(s.T(), vulnReqStore.UpsertMany(s.ctx, imageScopedReqs))
	require.NoError(s.T(), vulnReqStore.UpsertMany(s.ctx, globalScopedReqs))

	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}

	s.Require().NoError(migration.Run(dbs))

	objs, err := vulnReqStore.GetByQuery(s.ctx, search.EmptyQuery())
	assert.NoError(s.T(), err)
	assert.ElementsMatch(s.T(), ids, collectIDs(objs...))
	for _, obj := range objs {
		assert.Nil(s.T(), obj.GetScope().GetGlobalScope())
		assert.NotNil(s.T(), obj.GetScope().GetImageScope())
	}

	objs, err = vulnReqStore.GetByQuery(s.ctx,
		search.NewQueryBuilder().AddExactMatches(search.ImageRegistryScope, "reg-1").ProtoQuery())
	assert.NoError(s.T(), err)
	assert.ElementsMatch(s.T(), collectIDs(imageScopedReqs[0]), collectIDs(objs...))

	objs, err = vulnReqStore.GetByQuery(s.ctx,
		search.NewQueryBuilder().AddExactMatches(search.ImageRegistryScope, ".*").ProtoQuery())
	assert.NoError(s.T(), err)
	assert.ElementsMatch(s.T(), collectIDs(globalScopedReqs...), collectIDs(objs...))

	objs, err = vulnReqStore.GetByQuery(s.ctx,
		search.NewQueryBuilder().AddRegexes(search.ImageRemoteScope, ".*").ProtoQuery())
	assert.NoError(s.T(), err)
	assert.ElementsMatch(s.T(), ids, collectIDs(objs...))

	objs, err = vulnReqStore.GetByQuery(s.ctx,
		search.ConjunctionQuery(
			search.NewQueryBuilder().AddExactMatches(search.CVE, "cve-1").ProtoQuery(),
			search.DisjunctionQuery(
				search.NewQueryBuilder().AddExactMatches(search.ImageRegistryScope, "reg-1").ProtoQuery(),
				search.NewQueryBuilder().AddExactMatches(search.ImageRegistryScope, ".*").ProtoQuery(),
			),
		),
	)
	assert.NoError(s.T(), err)
	assert.ElementsMatch(s.T(), collectIDs(imageScopedReqs[0], globalScopedReqs[0], globalScopedReqs[2]), collectIDs(objs...))
}

func collectIDs(reqs ...*storage.VulnerabilityRequest) []string {
	var ids []string
	for _, req := range reqs {
		ids = append(ids, req.GetId())
	}
	return ids
}
