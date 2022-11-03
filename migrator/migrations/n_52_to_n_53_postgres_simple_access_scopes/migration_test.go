//go:build sql_integration

package n52ton53

// Code generation from pg-bindings generator disabled. To re-enable, check the gen.go file in
// central/role/store/permissionset/postgres
// central/role/store/role/postgres
// central/role/store/simpleaccessscope/postgres

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/n_52_to_n_53_postgres_simple_access_scopes/legacypermissionsets"
	"github.com/stackrox/rox/migrator/migrations/n_52_to_n_53_postgres_simple_access_scopes/legacyroles"
	"github.com/stackrox/rox/migrator/migrations/n_52_to_n_53_postgres_simple_access_scopes/legacysimpleaccessscopes"
	pgPermissionSetStore "github.com/stackrox/rox/migrator/migrations/n_52_to_n_53_postgres_simple_access_scopes/postgrespermissionsets"
	pgRoleStore "github.com/stackrox/rox/migrator/migrations/n_52_to_n_53_postgres_simple_access_scopes/postgresroles"
	pgSimpleAccessScopeStore "github.com/stackrox/rox/migrator/migrations/n_52_to_n_53_postgres_simple_access_scopes/postgressimpleaccessscopes"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
)

const (
	datasetSize     = 2500
	legacyBatchSize = 100
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(postgresMigrationSuite))
}

type postgresMigrationSuite struct {
	suite.Suite
	ctx context.Context

	legacyDB   *rocksdb.RocksDB
	postgresDB *pghelper.TestPostgres
}

var _ suite.TearDownTestSuite = (*postgresMigrationSuite)(nil)

func (s *postgresMigrationSuite) SetupTest() {
	s.T().Setenv(env.PostgresDatastoreEnabled.EnvVar(), "true")
	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		s.T().Skip("Skip postgres store tests")
		s.T().SkipNow()
	}

	var err error
	s.legacyDB, err = rocksdb.NewTemp(s.T().Name())
	s.NoError(err)

	s.Require().NoError(err)

	s.ctx = sac.WithAllAccess(context.Background())
	s.postgresDB = pghelper.ForT(s.T(), true)
}

func (s *postgresMigrationSuite) TearDownTest() {
	rocksdbtest.TearDownRocksDB(s.legacyDB)
	s.postgresDB.Teardown(s.T())
}

func (s *postgresMigrationSuite) TestSimpleAccessScopeMigration() {
	newStore := pgSimpleAccessScopeStore.New(s.postgresDB.Pool)
	legacyStore, err := legacysimpleaccessscopes.New(s.legacyDB)
	s.NoError(err)

	// Prepare data and write to legacy DB
	var simpleAccessScopes []*storage.SimpleAccessScope
	var simpleAccessScopesBatch []*storage.SimpleAccessScope
	batchID := 1
	for i := 0; i < datasetSize; i++ {
		simpleAccessScope := &storage.SimpleAccessScope{}
		s.NoError(testutils.FullInit(simpleAccessScope, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		simpleAccessScopes = append(simpleAccessScopes, simpleAccessScope)
		simpleAccessScopesBatch = append(simpleAccessScopesBatch, simpleAccessScope)
		if len(simpleAccessScopesBatch) >= legacyBatchSize {
			s.NoError(legacyStore.UpsertMany(s.ctx, simpleAccessScopesBatch))
			simpleAccessScopesBatch = simpleAccessScopesBatch[:0]
			batchID++
		}
	}

	s.NoError(legacyStore.UpsertMany(s.ctx, simpleAccessScopesBatch))

	// Move
	s.NoError(migrateAccessScopes(s.postgresDB.GetGormDB(), s.postgresDB.Pool, legacyStore))

	// Verify
	count, err := newStore.Count(s.ctx)
	s.NoError(err)
	s.Equal(len(simpleAccessScopes), count)
	for _, simpleAccessScope := range simpleAccessScopes {
		fetched, exists, err := newStore.Get(s.ctx, simpleAccessScope.GetId())
		s.NoError(err)
		s.True(exists)
		s.Equal(simpleAccessScope, fetched)
	}
}

func (s *postgresMigrationSuite) TestPermissionSetMigration() {
	newStore := pgPermissionSetStore.New(s.postgresDB.Pool)
	legacyStore, err := legacypermissionsets.New(s.legacyDB)
	s.NoError(err)

	// Prepare data and write to legacy DB
	var permissionSets []*storage.PermissionSet
	var permissionSetsBatch []*storage.PermissionSet
	batchID := 1
	for i := 0; i < datasetSize; i++ {
		permissionSet := &storage.PermissionSet{}
		s.NoError(testutils.FullInit(permissionSet, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		permissionSets = append(permissionSets, permissionSet)
		permissionSetsBatch = append(permissionSetsBatch, permissionSet)
		if len(permissionSetsBatch) >= legacyBatchSize {
			s.NoError(legacyStore.UpsertMany(s.ctx, permissionSetsBatch))
			permissionSetsBatch = permissionSetsBatch[:0]
			batchID++
		}
	}

	s.NoError(legacyStore.UpsertMany(s.ctx, permissionSetsBatch))

	// Move
	s.NoError(migratePermissionSets(s.postgresDB.GetGormDB(), s.postgresDB.Pool, legacyStore))

	// Verify
	count, err := newStore.Count(s.ctx)
	s.NoError(err)
	s.Equal(len(permissionSets), count)
	for _, permissionSet := range permissionSets {
		fetched, exists, err := newStore.Get(s.ctx, permissionSet.GetId())
		s.NoError(err)
		s.True(exists)
		s.Equal(permissionSet, fetched)
	}
}

func (s *postgresMigrationSuite) TestRoleMigration() {
	newStore := pgRoleStore.New(s.postgresDB.Pool)
	legacyStore, err := legacyroles.New(s.legacyDB)
	s.NoError(err)

	// Prepare data and write to legacy DB
	var roles []*storage.Role
	for i := 0; i < datasetSize; i++ {
		role := &storage.Role{}
		s.NoError(testutils.FullInit(role, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		roles = append(roles, role)
	}

	s.NoError(legacyStore.UpsertMany(s.ctx, roles))

	// Move
	s.NoError(migrateRoles(s.postgresDB.GetGormDB(), s.postgresDB.Pool, legacyStore))

	// Verify
	count, err := newStore.Count(s.ctx)
	s.NoError(err)
	s.Equal(len(roles), count)
	for _, role := range roles {
		fetched, exists, err := newStore.Get(s.ctx, role.GetName())
		s.NoError(err)
		s.True(exists)
		s.Equal(role, fetched)
	}
}
