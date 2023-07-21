//go:build sql_integration

package n52ton53

// Code generation from pg-bindings generator disabled. To re-enable, check the gen.go file in
// central/role/store/permissionset/postgres
// central/role/store/role/postgres
// central/role/store/simpleaccessscope/postgres

import (
	"context"
	"strings"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	frozenSchema "github.com/stackrox/rox/migrator/migrations/frozenschema/v73"
	"github.com/stackrox/rox/migrator/migrations/n_52_to_n_53_postgres_simple_access_scopes/legacypermissionsets"
	"github.com/stackrox/rox/migrator/migrations/n_52_to_n_53_postgres_simple_access_scopes/legacyroles"
	"github.com/stackrox/rox/migrator/migrations/n_52_to_n_53_postgres_simple_access_scopes/legacysimpleaccessscopes"
	pgPermissionSetStore "github.com/stackrox/rox/migrator/migrations/n_52_to_n_53_postgres_simple_access_scopes/postgrespermissionsets"
	pgReportConfigurationStore "github.com/stackrox/rox/migrator/migrations/n_52_to_n_53_postgres_simple_access_scopes/postgresreportconfigurations"
	pgRoleStore "github.com/stackrox/rox/migrator/migrations/n_52_to_n_53_postgres_simple_access_scopes/postgresroles"
	pgSimpleAccessScopeStore "github.com/stackrox/rox/migrator/migrations/n_52_to_n_53_postgres_simple_access_scopes/postgressimpleaccessscopes"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
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
	prefixlessNamedAccessScopeID       = "prefixlessAccessScopeID"
	prefixlessNamedAccessScopeName     = "Prefixless Named Access Scope"
	prefixlessUUIDAccessScopeID        = "07693a3d-ec29-4707-9ecf-e90fd0c2a338"
	prefixlessUUIDAccessScopeName      = "Prefixless UUID Access Scope"

	defaultAdminPermissionSetID      = permissionSetIDPrefix + "admin"
	defaultAdminPermissionSetName    = "Default Admin Permission Set"
	defaultAnalystPermissionSetID    = permissionSetIDPrefix + "analyst"
	defaultAnalystPermissionSetName  = "Default Analyst Permission Set"
	defaultNonePermissionSetID       = permissionSetIDPrefix + "none"
	defaultNonePermissionSetName     = "Default None Permission Set"
	prefixedNamedPermissionSetID     = permissionSetIDPrefix + "prefixedPermissionSetID"
	prefixedNamedPermissionSetName   = "Prefixed Named Permission Set"
	prefixedUUIDPermissionSetID      = permissionSetIDPrefix + "b450b538-2abc-41ae-ae2e-938dc7af3689"
	prefixedUUIDPermissionSetName    = "Prefixed UUID Permission Set"
	prefixlessNamedPermissionSetID   = "prefixlessPermissionSetID"
	prefixlessNamedPermissionSetName = "Prefixless Named Permission Set"
	prefixlessUUIDPermissionSetID    = "bc79bced-0fa5-45f6-9ae2-e054485ae6ff"
	prefixlessUUIDPermissionSetName  = "Prefixless UUID Permission Set"

	namePermission    = "name"
	uuidPermission    = "uuid"
	defaultPermission = "default"
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(postgresMigrationSuite))
}

type postgresMigrationSuite struct {
	suite.Suite
	ctx context.Context

	legacyDB   *rocksdb.RocksDB
	postgresDB *pghelper.TestPostgres
	gormDB     *gorm.DB
}

var _ suite.TearDownTestSuite = (*postgresMigrationSuite)(nil)

func (s *postgresMigrationSuite) SetupTest() {
	var err error
	s.legacyDB, err = rocksdb.NewTemp(s.T().Name())
	s.NoError(err)

	s.Require().NoError(err)

	s.ctx = sac.WithAllAccess(context.Background())
	s.postgresDB = pghelper.ForT(s.T(), false)
	s.gormDB = s.postgresDB.GetGormDB()
	pgutils.CreateTableFromModel(s.ctx, s.gormDB, frozenSchema.CreateTableReportConfigurationsStmt)
}

func (s *postgresMigrationSuite) TearDownTest() {
	pgtest.CloseGormDB(s.T(), s.gormDB)
	rocksdbtest.TearDownRocksDB(s.legacyDB)
	s.postgresDB.Teardown(s.T())
}

func (s *postgresMigrationSuite) TestSimpleAccessScopeMigration() {
	newStore := pgSimpleAccessScopeStore.New(s.postgresDB.DB)
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
	s.NoError(migrateAccessScopes(s.postgresDB.GetGormDB(), s.postgresDB.DB, legacyStore))

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
	newStore := pgPermissionSetStore.New(s.postgresDB.DB)
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
	s.NoError(migratePermissionSets(s.postgresDB.GetGormDB(), s.postgresDB.DB, legacyStore))

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
	newStore := pgRoleStore.New(s.postgresDB.DB)
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
	s.NoError(migrateRoles(s.postgresDB.GetGormDB(), s.postgresDB.DB, legacyStore))

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
	newScopeStore := pgSimpleAccessScopeStore.New(s.postgresDB.DB)
	newPermissionStore := pgPermissionSetStore.New(s.postgresDB.DB)
	newRoleStore := pgRoleStore.New(s.postgresDB.DB)
	postgresReportConfigStore := pgReportConfigurationStore.New(s.postgresDB.DB)
	legacyScopeStore, scopeErr := legacysimpleaccessscopes.New(s.legacyDB)
	s.NoError(scopeErr)
	legacyPermissionStore, permissionSetErr := legacypermissionsets.New(s.legacyDB)
	s.NoError(permissionSetErr)
	legacyRoleStore, roleErr := legacyroles.New(s.legacyDB)
	s.NoError(roleErr)

	// Prepare data and write to legacy DB
	accessScopes := map[string]*storage.SimpleAccessScope{
		prefixedNamedAccessScopeName: {
			Id:          prefixedNamedAccessScopeID,
			Name:        prefixedNamedAccessScopeName,
			Description: "Test access scope 1",
			Rules: &storage.SimpleAccessScope_Rules{
				NamespaceLabelSelectors: []*storage.SetBasedLabelSelector{
					{
						Requirements: []*storage.SetBasedLabelSelector_Requirement{
							{
								Key:    "k8s-app",
								Op:     storage.SetBasedLabelSelector_IN,
								Values: []string{"kube-dns"},
							},
						},
					},
				},
			},
		},
		prefixlessNamedAccessScopeName: {
			Id:          prefixlessNamedAccessScopeID,
			Name:        prefixlessNamedAccessScopeName,
			Description: "Test access scope 2",
			Rules: &storage.SimpleAccessScope_Rules{
				NamespaceLabelSelectors: []*storage.SetBasedLabelSelector{
					{
						Requirements: []*storage.SetBasedLabelSelector_Requirement{
							{
								Key:    "k8s-app",
								Op:     storage.SetBasedLabelSelector_NOT_IN,
								Values: []string{"kube-dns"},
							},
						},
					},
				},
			},
		},
		defaultUnrestrictedAccessScopeName: {
			Id:          defaultUnrestrictedAccessScopeID,
			Name:        defaultUnrestrictedAccessScopeName,
			Description: "Test access scope 3",
			Rules:       nil,
		},
		defaultDenyAllAccessScopeName: {
			Id:          defaultDenyAllAccessScopeID,
			Name:        defaultDenyAllAccessScopeName,
			Description: "Test access scope 4",
			Rules:       &storage.SimpleAccessScope_Rules{},
		},
		prefixedUUIDAccessScopeName: {
			Id:          prefixedUUIDAccessScopeID,
			Name:        prefixedUUIDAccessScopeName,
			Description: "Test access scope 5",
			Rules: &storage.SimpleAccessScope_Rules{
				IncludedClusters: []string{
					"3e86497c-1289-4752-9502-ae11b9f23027",
				},
			},
		},
		prefixlessUUIDAccessScopeName: {
			Id:          prefixlessUUIDAccessScopeID,
			Name:        prefixlessUUIDAccessScopeName,
			Description: "Test access scope 6",
			Rules: &storage.SimpleAccessScope_Rules{
				IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
					{
						ClusterName:   "TestCluster",
						NamespaceName: "TestNamespace",
					},
				},
			},
		},
	}
	accessScopeOldIDToNameMapping := make(map[string]string, len(accessScopes))
	accessScopeNameToNewIDMapping := make(map[string]string, len(accessScopes))
	scopesToInsert := make([]*storage.SimpleAccessScope, 0, len(accessScopes))
	for _, scope := range accessScopes {
		accessScopeOldIDToNameMapping[scope.GetId()] = scope.GetName()
		scopesToInsert = append(scopesToInsert, scope)
	}

	permissionSets := map[string]*storage.PermissionSet{
		prefixedNamedPermissionSetName: {
			Id:          prefixedNamedPermissionSetID,
			Name:        prefixedNamedPermissionSetName,
			Description: "Test permission set 1",
			ResourceToAccess: map[string]storage.Access{
				namePermission: storage.Access_READ_WRITE_ACCESS,
			},
		},
		prefixlessNamedPermissionSetName: {
			Id:          prefixlessNamedPermissionSetID,
			Name:        prefixlessNamedPermissionSetName,
			Description: "Test permission set 2",
			ResourceToAccess: map[string]storage.Access{
				namePermission: storage.Access_READ_ACCESS,
			},
		},
		defaultAdminPermissionSetName: {
			Id:          defaultAdminPermissionSetID,
			Name:        defaultAdminPermissionSetName,
			Description: "Test permission set 3",
			ResourceToAccess: map[string]storage.Access{
				defaultPermission: storage.Access_READ_WRITE_ACCESS,
				namePermission:    storage.Access_READ_WRITE_ACCESS,
				uuidPermission:    storage.Access_READ_WRITE_ACCESS,
			},
		},
		defaultAnalystPermissionSetName: {
			Id:          defaultAnalystPermissionSetID,
			Name:        defaultAnalystPermissionSetName,
			Description: "Test permission set 4",
			ResourceToAccess: map[string]storage.Access{
				defaultPermission: storage.Access_READ_ACCESS,
				namePermission:    storage.Access_READ_ACCESS,
				uuidPermission:    storage.Access_READ_ACCESS,
			},
		},
		defaultNonePermissionSetName: {
			Id:               defaultNonePermissionSetID,
			Name:             defaultNonePermissionSetName,
			Description:      "Test permission set 5",
			ResourceToAccess: nil,
		},
		prefixedUUIDPermissionSetName: {
			Id:          prefixedUUIDPermissionSetID,
			Name:        prefixedUUIDPermissionSetName,
			Description: "Test permission set 6",
			ResourceToAccess: map[string]storage.Access{
				uuidPermission: storage.Access_READ_WRITE_ACCESS,
			},
		},
		prefixlessUUIDPermissionSetName: {
			Id:          prefixlessUUIDPermissionSetID,
			Name:        prefixlessUUIDPermissionSetName,
			Description: "Test permission set 7",
			ResourceToAccess: map[string]storage.Access{
				uuidPermission: storage.Access_READ_ACCESS,
			},
		},
	}
	permissionSetOldIDToNameMapping := make(map[string]string, len(permissionSets))
	permissionSetNameToNewIDMapping := make(map[string]string, len(permissionSets))
	permissionSetsToInsert := make([]*storage.PermissionSet, 0, len(permissionSets))
	for _, permissionSet := range permissionSets {
		permissionSetOldIDToNameMapping[permissionSet.GetId()] = permissionSet.GetName()
		permissionSetsToInsert = append(permissionSetsToInsert, permissionSet)
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

	reportConfigurations := []*storage.ReportConfiguration{
		{
			Id:          uuid.NewV4().String(),
			Name:        "Report1",
			Description: "Report with prefixed named scope",
			Type:        storage.ReportConfiguration_VULNERABILITY,
			ScopeId:     prefixedNamedAccessScopeID,
		},
		{
			Id:          uuid.NewV4().String(),
			Name:        "Report2",
			Description: "Report with prefixless named scope",
			Type:        storage.ReportConfiguration_VULNERABILITY,
			ScopeId:     prefixlessNamedAccessScopeID,
		},
		{
			Id:          uuid.NewV4().String(),
			Name:        "Report3",
			Description: "Report with default unrestricted scope",
			Type:        storage.ReportConfiguration_VULNERABILITY,
			ScopeId:     defaultUnrestrictedAccessScopeID,
		},
		{
			Id:          uuid.NewV4().String(),
			Name:        "Report4",
			Description: "Report with default deny all scope",
			Type:        storage.ReportConfiguration_VULNERABILITY,
			ScopeId:     defaultDenyAllAccessScopeID,
		},
		{
			Id:          uuid.NewV4().String(),
			Name:        "Report5",
			Description: "Report with prefixed UUID scope",
			Type:        storage.ReportConfiguration_VULNERABILITY,
			ScopeId:     prefixedUUIDAccessScopeID,
		},
		{
			Id:          uuid.NewV4().String(),
			Name:        "Report6",
			Description: "Report with prefixless UUID scope",
			Type:        storage.ReportConfiguration_VULNERABILITY,
			ScopeId:     prefixlessUUIDAccessScopeID,
		},
	}

	s.NoError(legacyScopeStore.UpsertMany(s.ctx, scopesToInsert))
	s.NoError(legacyPermissionStore.UpsertMany(s.ctx, permissionSetsToInsert))
	s.NoError(legacyRoleStore.UpsertMany(s.ctx, roles))
	s.NoError(postgresReportConfigStore.UpsertMany(s.ctx, reportConfigurations))

	// Move
	s.NoError(migrateAll(s.legacyDB, s.postgresDB.GetGormDB(), s.postgresDB.DB))

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
		// Ensure the newly generated ID is a UUID
		_, identifierParseErr := uuid.FromString(scopeID)
		s.NoError(identifierParseErr)
		// Check the migrated access scope matches the original one
		fetched, exists, err := newScopeStore.Get(s.ctx, scopeID)
		s.NoError(err)
		s.True(exists)
		if fetched != nil {
			accessScopeNameToNewIDMapping[fetched.GetName()] = fetched.GetId()
		}
		if fetched.GetName() == prefixedUUIDAccessScopeName {
			s.Equal(strings.TrimPrefix(prefixedUUIDAccessScopeID, accessScopeIDPrefix), scopeID)
		}
		if fetched.GetName() == prefixlessUUIDAccessScopeName {
			s.Equal(prefixlessUUIDAccessScopeID, scopeID)
		}
		referenceScope := accessScopes[fetched.GetName()]
		s.Equal(referenceScope.GetDescription(), fetched.GetDescription())
		s.Equal(referenceScope.GetRules(), fetched.GetRules())
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
		// Check the new allocated ID is a UUID
		_, identifierParseErr := uuid.FromString(permissionSetID)
		s.NoError(identifierParseErr)
		// Validate the retrieved permission set matches the initial one
		fetched, exists, err := newPermissionStore.Get(s.ctx, permissionSetID)
		s.NoError(err)
		s.True(exists)
		if fetched != nil {
			permissionSetNameToNewIDMapping[fetched.GetName()] = fetched.GetId()
		}
		if fetched.GetName() == prefixedUUIDPermissionSetName {
			s.Equal(strings.TrimPrefix(prefixedUUIDPermissionSetID, permissionSetIDPrefix), permissionSetID)
		}
		if fetched.GetName() == prefixlessUUIDPermissionSetName {
			s.Equal(prefixlessUUIDPermissionSetID, permissionSetID)
		}
		referencePermissionSet := permissionSets[fetched.GetName()]
		s.Equal(referencePermissionSet.GetDescription(), fetched.GetDescription())
		s.Equal(referencePermissionSet.GetResourceToAccess(), fetched.GetResourceToAccess())
	}
	roleCount, err := newRoleStore.Count(s.ctx)
	s.NoError(err)
	s.Equal(len(roles), roleCount)
	for _, role := range roles {
		fetched, exists, err := newRoleStore.Get(s.ctx, role.GetName())
		s.NoError(err)
		s.True(exists)
		expectedRole := role.Clone()
		// Map role permission set ID to new ID
		permissionSetName := permissionSetOldIDToNameMapping[role.GetPermissionSetId()]
		expectedRole.PermissionSetId = permissionSetNameToNewIDMapping[permissionSetName]
		// Map role access scope ID to new ID
		accessScopeName := accessScopeOldIDToNameMapping[role.GetAccessScopeId()]
		expectedRole.AccessScopeId = accessScopeNameToNewIDMapping[accessScopeName]
		s.Equal(expectedRole, fetched)
	}
	reportConfigurationCount, err := postgresReportConfigStore.Count(s.ctx)
	s.NoError(err)
	s.Equal(len(reportConfigurations), reportConfigurationCount)
	for _, reportConfiguration := range reportConfigurations {
		fetched, exists, err := postgresReportConfigStore.Get(s.ctx, reportConfiguration.GetId())
		s.NoError(err)
		s.True(exists)
		expectedReportConfiguration := reportConfiguration.Clone()
		scopeName := accessScopeOldIDToNameMapping[reportConfiguration.GetScopeId()]
		expectedReportConfiguration.ScopeId = accessScopeNameToNewIDMapping[scopeName]
		s.Equal(expectedReportConfiguration, fetched)
	}
}
