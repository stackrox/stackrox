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

	defaultDenyAllAccessScopeID        = accessScopeIDPrefix + "denyall"
	defaultDenyAllAccessScopeName      = "Default DenyAll Access Scope"
	defaultUnrestrictedAccessScopeID   = accessScopeIDPrefix + "unrestricted"
	defaultUnrestrictedAccessScopeName = "Default Unrestricted Access Scope"
	prefixedNamedAccessScopeID         = accessScopeIDPrefix + "prefixedAccessScopeID"
	prefixedNamedAccessScopeName       = "Prefixed Named Access Scope"
	prefixedUUIDAccessScopeID          = accessScopeIDPrefix + "47d9f01d-3def-4f0d-9777-916b5879aaf7"
	prefixedUUIDAccessScopeName        = "Prefixed UUID Access Scope"
	prefixlessUUIDAccessScopeID        = "07693a3d-ec29-4707-9ecf-e90fd0c2a338"
	prefixlessUUIDAccessScopeName      = "Prefixless Named Access Scope"
	prefixlessNamedAccessScopeID       = "prefixlessAccessScopeID"
	prefixlessNamedAccessScopeName     = "Prefixless Named Access Scope"

	prefixedNamedPermissionSetID     = permissionSetIDPrefix + "prefixedPermissionSetID"
	prefixedNamedPermissionSetName   = "Prefixed Named Permission Set"
	prefixlessNamedPermissionSetID   = "prefixlessPermissionSetID"
	prefixlessNamedPermissionSetName = "Prefixless Named Permission Set"
	defaultAdminPermissionSetID      = permissionSetIDPrefix + "admin"
	defaultAdminPermissionSetName    = "Default Admin Permission Set"
	defaultAnalystPermissionSetID    = permissionSetIDPrefix + "analyst"
	defaultAnalystPermissionSetName  = "Default Analyst Permission Set"
	defaultNonePermissionSetID       = permissionSetIDPrefix + "none"
	defaultNonePermissionSetName     = "Default None Permission Set"
	prefixedUUIDPermissionSetID      = permissionSetIDPrefix + "b450b538-2abc-41ae-ae2e-938dc7af3689"
	prefixedUUIDPermissionSetName    = "Prefixed UUID Permission Set"
	prefixlessUUIDPermissionSetID    = "bc79bced-0fa5-45f6-9ae2-e054485ae6ff"
	prefixlessUUIDPermissionSetName  = "Prefixless Named Permission Set"
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

func (s *postgresMigrationSuite) TestMigrateAll() {
	newScopeStore := pgSimpleAccessScopeStore.New(s.postgresDB.Pool)
	newPermissionStore := pgPermissionSetStore.New(s.postgresDB.Pool)
	newRoleStore := pgRoleStore.New(s.postgresDB.Pool)
	legacyScopeStore, scopeErr := legacysimpleaccessscopes.New(s.legacyDB)
	s.NoError(scopeErr)
	legacyPermissionStore, permissionSetErr := legacypermissionsets.New(s.legacyDB)
	s.NoError(permissionSetErr)
	legacyRoleStore, roleErr := legacyroles.New(s.legacyDB)
	s.NoError(roleErr)

	// Prepare data and write to legacy DB
	accessScopes := []*storage.SimpleAccessScope{
		{
			Id:          prefixedNamedAccessScopeID,
			Name:        prefixedNamedAccessScopeName,
			Description: "",
			Rules:       nil,
		},
		{
			Id:          prefixlessNamedAccessScopeID,
			Name:        prefixlessNamedAccessScopeName,
			Description: "",
			Rules:       nil,
		},
		{
			Id:          defaultUnrestrictedAccessScopeID,
			Name:        defaultUnrestrictedAccessScopeName,
			Description: "",
			Rules:       nil,
		},
		{
			Id:          defaultDenyAllAccessScopeID,
			Name:        defaultDenyAllAccessScopeName,
			Description: "",
			Rules:       nil,
		},
		{
			Id:          prefixedUUIDAccessScopeID,
			Name:        prefixedUUIDAccessScopeName,
			Description: "",
			Rules:       nil,
		},
		{
			Id:          prefixlessUUIDAccessScopeID,
			Name:        prefixlessUUIDAccessScopeName,
			Description: "",
			Rules:       nil,
		},
	}
	accessScopeOldIDToNameMapping := make(map[string]string, len(accessScopes))
	accessScopeNameToNewIDMapping := make(map[string]string, len(accessScopes))
	for _, scope := range accessScopes {
		accessScopeOldIDToNameMapping[scope.GetId()] = scope.GetName()
	}

	permissionSets := []*storage.PermissionSet{
		{
			Id:               prefixedNamedPermissionSetID,
			Name:             prefixedNamedPermissionSetName,
			Description:      "",
			ResourceToAccess: nil,
		},
		{
			Id:               prefixlessNamedPermissionSetID,
			Name:             prefixlessNamedPermissionSetName,
			Description:      "",
			ResourceToAccess: nil,
		},
		{
			Id:               defaultAdminPermissionSetID,
			Name:             defaultAdminPermissionSetName,
			Description:      "",
			ResourceToAccess: nil,
		},
		{
			Id:               defaultAnalystPermissionSetID,
			Name:             defaultAnalystPermissionSetName,
			Description:      "",
			ResourceToAccess: nil,
		},
		{
			Id:               defaultNonePermissionSetID,
			Name:             defaultNonePermissionSetName,
			Description:      "",
			ResourceToAccess: nil,
		},
		{
			Id:               prefixedUUIDPermissionSetID,
			Name:             prefixedUUIDPermissionSetName,
			Description:      "",
			ResourceToAccess: nil,
		},
		{
			Id:               prefixlessUUIDPermissionSetID,
			Name:             prefixlessUUIDPermissionSetName,
			Description:      "",
			ResourceToAccess: nil,
		},
	}
	permissionSetOldIDToNameMapping := make(map[string]string, len(permissionSets))
	permissionSetNameToNewIDMapping := make(map[string]string, len(permissionSets))
	for _, permissionSet := range permissionSets {
		permissionSetOldIDToNameMapping[permissionSet.GetId()] = permissionSet.GetName()
	}

	roles := []*storage.Role{
		{
			Name:            "R11",
			Description:     "",
			PermissionSetId: prefixedNamedPermissionSetID,
			AccessScopeId:   prefixedNamedAccessScopeID,
		},
		{
			Name:            "R12",
			Description:     "",
			PermissionSetId: prefixedNamedPermissionSetID,
			AccessScopeId:   prefixlessNamedAccessScopeID,
		},
		{
			Name:            "R13",
			Description:     "",
			PermissionSetId: prefixedNamedPermissionSetID,
			AccessScopeId:   defaultDenyAllAccessScopeID,
		},
		{
			Name:            "R14",
			Description:     "",
			PermissionSetId: prefixedNamedPermissionSetID,
			AccessScopeId:   defaultUnrestrictedAccessScopeID,
		},
		{
			Name:            "R15",
			Description:     "",
			PermissionSetId: prefixedNamedPermissionSetID,
			AccessScopeId:   prefixedUUIDAccessScopeID,
		},
		{
			Name:            "R16",
			Description:     "",
			PermissionSetId: prefixedNamedPermissionSetID,
			AccessScopeId:   prefixlessUUIDAccessScopeID,
		},
		{
			Name:            "R21",
			Description:     "",
			PermissionSetId: prefixlessNamedPermissionSetID,
			AccessScopeId:   prefixedNamedAccessScopeID,
		},
		{
			Name:            "R22",
			Description:     "",
			PermissionSetId: prefixlessNamedPermissionSetID,
			AccessScopeId:   prefixlessNamedAccessScopeID,
		},
		{
			Name:            "R23",
			Description:     "",
			PermissionSetId: prefixlessNamedPermissionSetID,
			AccessScopeId:   defaultDenyAllAccessScopeID,
		},
		{
			Name:            "R24",
			Description:     "",
			PermissionSetId: prefixlessNamedPermissionSetID,
			AccessScopeId:   defaultUnrestrictedAccessScopeID,
		},
		{
			Name:            "R25",
			Description:     "",
			PermissionSetId: prefixlessNamedPermissionSetID,
			AccessScopeId:   prefixedUUIDAccessScopeID,
		},
		{
			Name:            "R26",
			Description:     "",
			PermissionSetId: prefixlessNamedPermissionSetID,
			AccessScopeId:   prefixlessUUIDAccessScopeID,
		},
		{
			Name:            "R31",
			Description:     "",
			PermissionSetId: defaultAdminPermissionSetID,
			AccessScopeId:   prefixedNamedAccessScopeID,
		},
		{
			Name:            "R32",
			Description:     "",
			PermissionSetId: defaultAdminPermissionSetID,
			AccessScopeId:   prefixlessNamedAccessScopeID,
		},
		{
			Name:            "R33",
			Description:     "",
			PermissionSetId: defaultAdminPermissionSetID,
			AccessScopeId:   defaultDenyAllAccessScopeID,
		},
		{
			Name:            "R34",
			Description:     "",
			PermissionSetId: defaultAdminPermissionSetID,
			AccessScopeId:   defaultUnrestrictedAccessScopeID,
		},
		{
			Name:            "R35",
			Description:     "",
			PermissionSetId: defaultAdminPermissionSetID,
			AccessScopeId:   prefixedUUIDAccessScopeID,
		},
		{
			Name:            "R36",
			Description:     "",
			PermissionSetId: defaultAdminPermissionSetID,
			AccessScopeId:   prefixlessUUIDAccessScopeID,
		},
		{
			Name:            "R41",
			Description:     "",
			PermissionSetId: defaultAnalystPermissionSetID,
			AccessScopeId:   prefixedNamedAccessScopeID,
		},
		{
			Name:            "R42",
			Description:     "",
			PermissionSetId: defaultAnalystPermissionSetID,
			AccessScopeId:   prefixlessNamedAccessScopeID,
		},
		{
			Name:            "R43",
			Description:     "",
			PermissionSetId: defaultAnalystPermissionSetID,
			AccessScopeId:   defaultDenyAllAccessScopeID,
		},
		{
			Name:            "R44",
			Description:     "",
			PermissionSetId: defaultAnalystPermissionSetID,
			AccessScopeId:   defaultUnrestrictedAccessScopeID,
		},
		{
			Name:            "R45",
			Description:     "",
			PermissionSetId: defaultAnalystPermissionSetID,
			AccessScopeId:   prefixedUUIDAccessScopeID,
		},
		{
			Name:            "R46",
			Description:     "",
			PermissionSetId: defaultAnalystPermissionSetID,
			AccessScopeId:   prefixlessUUIDAccessScopeID,
		},
		{
			Name:            "R51",
			Description:     "",
			PermissionSetId: defaultNonePermissionSetID,
			AccessScopeId:   prefixedNamedAccessScopeID,
		},
		{
			Name:            "R52",
			Description:     "",
			PermissionSetId: defaultNonePermissionSetID,
			AccessScopeId:   prefixlessNamedAccessScopeID,
		},
		{
			Name:            "R53",
			Description:     "",
			PermissionSetId: defaultNonePermissionSetID,
			AccessScopeId:   defaultDenyAllAccessScopeID,
		},
		{
			Name:            "R54",
			Description:     "",
			PermissionSetId: defaultNonePermissionSetID,
			AccessScopeId:   defaultUnrestrictedAccessScopeID,
		},
		{
			Name:            "R55",
			Description:     "",
			PermissionSetId: defaultNonePermissionSetID,
			AccessScopeId:   prefixedUUIDAccessScopeID,
		},
		{
			Name:            "R56",
			Description:     "",
			PermissionSetId: defaultNonePermissionSetID,
			AccessScopeId:   prefixlessUUIDAccessScopeID,
		},
		{
			Name:            "R61",
			Description:     "",
			PermissionSetId: prefixedUUIDPermissionSetID,
			AccessScopeId:   prefixedNamedAccessScopeID,
		},
		{
			Name:            "R62",
			Description:     "",
			PermissionSetId: prefixedUUIDPermissionSetID,
			AccessScopeId:   prefixlessNamedAccessScopeID,
		},
		{
			Name:            "R63",
			Description:     "",
			PermissionSetId: prefixedUUIDPermissionSetID,
			AccessScopeId:   defaultDenyAllAccessScopeID,
		},
		{
			Name:            "R64",
			Description:     "",
			PermissionSetId: prefixedUUIDPermissionSetID,
			AccessScopeId:   defaultUnrestrictedAccessScopeID,
		},
		{
			Name:            "R65",
			Description:     "",
			PermissionSetId: prefixedUUIDPermissionSetID,
			AccessScopeId:   prefixedUUIDAccessScopeID,
		},
		{
			Name:            "R66",
			Description:     "",
			PermissionSetId: prefixedUUIDPermissionSetID,
			AccessScopeId:   prefixlessUUIDAccessScopeID,
		},
		{
			Name:            "R71",
			Description:     "",
			PermissionSetId: prefixlessUUIDPermissionSetID,
			AccessScopeId:   prefixedNamedAccessScopeID,
		},
		{
			Name:            "R72",
			Description:     "",
			PermissionSetId: prefixlessUUIDPermissionSetID,
			AccessScopeId:   prefixlessNamedAccessScopeID,
		},
		{
			Name:            "R73",
			Description:     "",
			PermissionSetId: prefixlessUUIDPermissionSetID,
			AccessScopeId:   defaultDenyAllAccessScopeID,
		},
		{
			Name:            "R74",
			Description:     "",
			PermissionSetId: prefixlessUUIDPermissionSetID,
			AccessScopeId:   defaultUnrestrictedAccessScopeID,
		},
		{
			Name:            "R75",
			Description:     "",
			PermissionSetId: prefixlessUUIDPermissionSetID,
			AccessScopeId:   prefixedUUIDAccessScopeID,
		},
		{
			Name:            "R76",
			Description:     "",
			PermissionSetId: prefixlessUUIDPermissionSetID,
			AccessScopeId:   prefixlessUUIDAccessScopeID,
		},
	}

	// Roles : all possible pair combinations of the above

	s.NoError(legacyScopeStore.UpsertMany(s.ctx, accessScopes))
	s.NoError(legacyPermissionStore.UpsertMany(s.ctx, permissionSets))
	s.NoError(legacyRoleStore.UpsertMany(s.ctx, roles))

	// Move
	s.NoError(migrateAll(s.legacyDB, s.postgresDB.GetGormDB(), s.postgresDB.Pool))

	// Verify
	scopeCount, err := newScopeStore.Count(s.ctx)
	s.NoError(err)
	s.Equal(len(accessScopes), scopeCount)
	newScopeIDs := make([]string, 0, len(accessScopes))
	scopeWalkErr := newScopeStore.Walk(s.ctx, func(obj *storage.SimpleAccessScope) error {
		newScopeIDs = append(newScopeIDs, obj.GetId())
		return nil
	})
	s.NoError(scopeWalkErr)
	for _, scopeID := range newScopeIDs {
		fetched, exists, err := newScopeStore.Get(s.ctx, scopeID)
		s.NoError(err)
		s.True(exists)
		if fetched != nil {
			accessScopeNameToNewIDMapping[fetched.GetName()] = fetched.GetId()
		}
	}
	s.Equal(len(accessScopeOldIDToNameMapping), len(accessScopeNameToNewIDMapping))
	permissionSetCount, err := newPermissionStore.Count(s.ctx)
	s.NoError(err)
	s.Equal(len(permissionSets), permissionSetCount)
	newPermissionSetIDs := make([]string, 0, len(permissionSets))
	permissionWalkErr := newPermissionStore.Walk(s.ctx, func(obj *storage.PermissionSet) error {
		newPermissionSetIDs = append(newPermissionSetIDs, obj.GetId())
		return nil
	})
	s.NoError(permissionWalkErr)
	for _, permissionSetID := range newPermissionSetIDs {
		fetched, exists, err := newPermissionStore.Get(s.ctx, permissionSetID)
		s.NoError(err)
		s.True(exists)
		if fetched != nil {
			permissionSetNameToNewIDMapping[fetched.GetName()] = fetched.GetId()
		}
	}
	roleCount, err := newRoleStore.Count(s.ctx)
	s.NoError(err)
	s.Equal(len(roles), roleCount)
	for _, role := range roles {
		fetched, exists, err := newRoleStore.Get(s.ctx, role.GetName())
		s.NoError(err)
		s.True(exists)
		expectedRole := role.Clone()
		permissionSetName := permissionSetOldIDToNameMapping[role.GetPermissionSetId()]
		expectedRole.PermissionSetId = permissionSetNameToNewIDMapping[permissionSetName]
		accessScopeName := accessScopeOldIDToNameMapping[role.GetAccessScopeId()]
		expectedRole.AccessScopeId = accessScopeNameToNewIDMapping[accessScopeName]
		s.Equal(expectedRole, fetched)
	}
}
