//go:build sql_integration

package m182tom183

import (
	"context"
	"testing"
	"time"

	protobufTypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	apiTokenStore "github.com/stackrox/rox/migrator/migrations/m_182_to_m_183_remove_default_scope_manager_role/apitokenstore"
	groupStore "github.com/stackrox/rox/migrator/migrations/m_182_to_m_183_remove_default_scope_manager_role/groupstore"
	permissionSetStore "github.com/stackrox/rox/migrator/migrations/m_182_to_m_183_remove_default_scope_manager_role/permissionsetstore"
	roleStore "github.com/stackrox/rox/migrator/migrations/m_182_to_m_183_remove_default_scope_manager_role/rolestore"
	frozenSchema "github.com/stackrox/rox/migrator/migrations/m_182_to_m_183_remove_default_scope_manager_role/schema"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

var (
	ctx = sac.WithAllAccess(context.Background())
)

type migrationTestSuite struct {
	suite.Suite

	db *pghelper.TestPostgres
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(migrationTestSuite))
}

func (s *migrationTestSuite) SetupTest() {
	s.db = pghelper.ForT(s.T(), false)

	pgutils.CreateTableFromModel(ctx, s.db.GetGormDB(), frozenSchema.CreateTableAPITokensStmt)
	pgutils.CreateTableFromModel(ctx, s.db.GetGormDB(), frozenSchema.CreateTableGroupsStmt)
	pgutils.CreateTableFromModel(ctx, s.db.GetGormDB(), frozenSchema.CreateTablePermissionSetsStmt)
	pgutils.CreateTableFromModel(ctx, s.db.GetGormDB(), frozenSchema.CreateTableRolesStmt)
}

func (s *migrationTestSuite) TearDownTest() {
	s.db.Teardown(s.T())
}

const (
	deployment = "Deployment"
)

var (
	defaultScopeManagerRole = &storage.Role{
		Name:            scopeManagerObjectName,
		Description:     oldScopeManagerDescription,
		PermissionSetId: scopeManagerPermissionSetID,
		AccessScopeId:   unrestrictedAccessScopeID,
		Traits: &storage.Traits{
			Origin: storage.Traits_DEFAULT,
		},
	}

	defaultScopeManagerPermissionSet = &storage.PermissionSet{
		Id:          scopeManagerPermissionSetID,
		Name:        scopeManagerObjectName,
		Description: oldScopeManagerDescription,
		ResourceToAccess: map[string]storage.Access{
			access:    storage.Access_READ_ACCESS,
			cluster:   storage.Access_READ_ACCESS,
			namespace: storage.Access_READ_ACCESS,
		},
		Traits: &storage.Traits{
			Origin: storage.Traits_DEFAULT,
		},
	}

	migratedScopeManagerRole = &storage.Role{
		Name:            scopeManagerObjectName,
		Description:     oldScopeManagerDescription + updatedDescriptionSuffix,
		PermissionSetId: scopeManagerPermissionSetID,
		AccessScopeId:   unrestrictedAccessScopeID,
		Traits:          imperativeObjectTraits,
	}

	migratedScopeManagerPermissionSet = &storage.PermissionSet{
		Id:          scopeManagerPermissionSetID,
		Name:        scopeManagerObjectName,
		Description: oldScopeManagerDescription + updatedDescriptionSuffix,
		ResourceToAccess: map[string]storage.Access{
			access:    storage.Access_READ_WRITE_ACCESS,
			cluster:   storage.Access_READ_ACCESS,
			namespace: storage.Access_READ_ACCESS,
		},
		Traits: imperativeObjectTraits,
	}

	otherPermissionSet = &storage.PermissionSet{
		Id:          "12345678-9ABC-DEF0-AAAA-111111111111",
		Name:        "Test Permission Set",
		Description: "Some test PermissionSet",
		ResourceToAccess: map[string]storage.Access{
			access:     storage.Access_READ_WRITE_ACCESS,
			cluster:    storage.Access_READ_ACCESS,
			deployment: storage.Access_READ_ACCESS,
			namespace:  storage.Access_READ_ACCESS,
		},
		Traits: imperativeObjectTraits,
	}

	testRoleNoRef = &storage.Role{
		Name:            "Test Role 1",
		Description:     "A role for testing purpose",
		PermissionSetId: "12345678-9ABC-DEF0-8888-FFFFFFFFFFFF",
		AccessScopeId:   unrestrictedAccessScopeID,
		Traits:          imperativeObjectTraits,
	}

	testRoleWithReference = &storage.Role{
		Name:            "Test Role 2",
		Description:     "Another role for testing purpose",
		PermissionSetId: scopeManagerPermissionSetID,
		AccessScopeId:   unrestrictedAccessScopeID,
		Traits:          imperativeObjectTraits,
	}

	testAPITokenNoRef = &storage.TokenMetadata{
		Id:         "12345678-9ABC-DEF0-AAAA-222222222222",
		Name:       "Test Token 1",
		Roles:      []string{"Admin", "Vulnerability Management Requester"},
		IssuedAt:   getTime("17/09/1991"),
		Expiration: getTime("17/09/1992"),
		Revoked:    false,
	}

	testAPITokenWithReference = &storage.TokenMetadata{
		Id:         "12345678-9ABC-DEF0-AAAA-333333333333",
		Name:       "Test Token 2",
		Roles:      []string{scopeManagerObjectName, "Vulnerability Management Requester"},
		IssuedAt:   getTime("17/12/2003"),
		Expiration: getTime("17/12/2004"),
		Revoked:    false,
	}

	testGroupNoRef = &storage.Group{
		Props: &storage.GroupProperties{
			Id: "12345678-9ABC-DEF0-CCCC-111111111111",
			Traits: &storage.Traits{
				Origin: storage.Traits_IMPERATIVE,
			},
			AuthProviderId: "12345678-9ABC-DEF0-EEEE-FFFFFFFFFFFF",
			Key:            "Couch",
			Value:          "Base",
		},
		RoleName: "Admin",
	}

	testGroupWithReference = &storage.Group{
		Props: &storage.GroupProperties{
			Id: "12345678-9ABC-DEF0-CCCC-222222222222",
			Traits: &storage.Traits{
				Origin: storage.Traits_IMPERATIVE,
			},
			AuthProviderId: "12345678-9ABC-DEF0-EEEE-EEEEEEEEEEEE",
			Key:            "Mem",
			Value:          "Cache",
		},
		RoleName: scopeManagerObjectName,
	}
)

func getTime(formattedValue string) *protobufTypes.Timestamp {
	t, _ := time.Parse("DD/MM/YYYY", formattedValue)
	return protoconv.ConvertTimeToTimestamp(t)
}

func (s *migrationTestSuite) TestMigrationNoReference() {
	inputAPITokens := []*storage.TokenMetadata{
		testAPITokenNoRef,
	}
	inputGroups := []*storage.Group{
		testGroupNoRef,
	}
	inputPermissionSets := []*storage.PermissionSet{
		defaultScopeManagerPermissionSet,
		otherPermissionSet,
	}
	inputRoles := []*storage.Role{
		defaultScopeManagerRole,
		testRoleNoRef,
	}

	// No change.
	expectedAPITokens := []*storage.TokenMetadata{
		testAPITokenNoRef,
	}
	// No change.
	expectedGroups := []*storage.Group{
		testGroupNoRef,
	}
	// No reference to default Scope Manager permission set (except from default role)
	// leads to permission set deletion.
	expectedPermissionSets := []*storage.PermissionSet{
		otherPermissionSet,
	}
	// No reference to default Scope Manager role leads to role deletion.
	expectedRoles := []*storage.Role{
		testRoleNoRef,
	}

	s.testDataSetMigration(
		inputAPITokens,
		inputGroups,
		inputPermissionSets,
		inputRoles,
		expectedAPITokens,
		expectedGroups,
		expectedPermissionSets,
		expectedRoles,
	)
}

func (s *migrationTestSuite) TestMigrationPermissionSetReferenceOnly() {
	inputAPITokens := []*storage.TokenMetadata{
		testAPITokenNoRef,
	}
	inputGroups := []*storage.Group{
		testGroupNoRef,
	}
	inputPermissionSets := []*storage.PermissionSet{
		defaultScopeManagerPermissionSet,
		otherPermissionSet,
	}
	inputRoles := []*storage.Role{
		defaultScopeManagerRole,
		testRoleNoRef,
		testRoleWithReference,
	}

	// No change.
	expectedAPITokens := []*storage.TokenMetadata{
		testAPITokenNoRef,
	}
	// No change.
	expectedGroups := []*storage.Group{
		testGroupNoRef,
	}
	// Reference from any role (other than default Scope Manager role) to the Scope Manager permission set
	// leads to permission set migration.
	expectedPermissionSets := []*storage.PermissionSet{
		migratedScopeManagerPermissionSet,
		otherPermissionSet,
	}
	// No reference to default Scope Manager role leads to role deletion.
	expectedRoles := []*storage.Role{
		testRoleNoRef,
		testRoleWithReference,
	}

	s.testDataSetMigration(
		inputAPITokens,
		inputGroups,
		inputPermissionSets,
		inputRoles,
		expectedAPITokens,
		expectedGroups,
		expectedPermissionSets,
		expectedRoles,
	)
}

func (s *migrationTestSuite) TestMigrationRoleReferenceFromGroup() {
	inputAPITokens := []*storage.TokenMetadata{
		testAPITokenNoRef,
	}
	inputGroups := []*storage.Group{
		testGroupNoRef,
		testGroupWithReference,
	}
	inputPermissionSets := []*storage.PermissionSet{
		defaultScopeManagerPermissionSet,
		otherPermissionSet,
	}
	inputRoles := []*storage.Role{
		defaultScopeManagerRole,
		testRoleNoRef,
	}

	// No change.
	expectedAPITokens := []*storage.TokenMetadata{
		testAPITokenNoRef,
	}
	// No change.
	expectedGroups := []*storage.Group{
		testGroupNoRef,
		testGroupWithReference,
	}
	// As the Scope Manager role is referenced, the permission set is kept in its updated form.
	expectedPermissionSets := []*storage.PermissionSet{
		migratedScopeManagerPermissionSet,
		otherPermissionSet,
	}
	// As the Scope Manager role is referenced, it is replaced by the updated role.
	expectedRoles := []*storage.Role{
		migratedScopeManagerRole,
		testRoleNoRef,
	}

	s.testDataSetMigration(
		inputAPITokens,
		inputGroups,
		inputPermissionSets,
		inputRoles,
		expectedAPITokens,
		expectedGroups,
		expectedPermissionSets,
		expectedRoles,
	)
}

func (s *migrationTestSuite) TestMigrationRoleReferenceFromAPIToken() {
	inputAPITokens := []*storage.TokenMetadata{
		testAPITokenNoRef,
		testAPITokenWithReference,
	}
	inputGroups := []*storage.Group{
		testGroupNoRef,
	}
	inputPermissionSets := []*storage.PermissionSet{
		defaultScopeManagerPermissionSet,
		otherPermissionSet,
	}
	inputRoles := []*storage.Role{
		defaultScopeManagerRole,
		testRoleNoRef,
	}

	// No change.
	expectedAPITokens := []*storage.TokenMetadata{
		testAPITokenNoRef,
		testAPITokenWithReference,
	}
	// No change.
	expectedGroups := []*storage.Group{
		testGroupNoRef,
	}
	// As the Scope Manager role is referenced, the permission set is kept in its updated form.
	expectedPermissionSets := []*storage.PermissionSet{
		migratedScopeManagerPermissionSet,
		otherPermissionSet,
	}
	// As the Scope Manager role is referenced, it is replaced by the updated role.
	expectedRoles := []*storage.Role{
		migratedScopeManagerRole,
		testRoleNoRef,
	}

	s.testDataSetMigration(
		inputAPITokens,
		inputGroups,
		inputPermissionSets,
		inputRoles,
		expectedAPITokens,
		expectedGroups,
		expectedPermissionSets,
		expectedRoles,
	)
}

func (s *migrationTestSuite) testDataSetMigration(
	inputAPITokens []*storage.TokenMetadata,
	inputGroups []*storage.Group,
	inputPermissionSets []*storage.PermissionSet,
	inputRoles []*storage.Role,
	expectedAPITokens []*storage.TokenMetadata,
	expectedGroups []*storage.Group,
	expectedPermissionSets []*storage.PermissionSet,
	expectedRoles []*storage.Role,
) {
	ctx := sac.WithAllAccess(context.Background())

	// Instantiate Stores
	apiTokenStorage := apiTokenStore.New(s.db)
	groupStorage := groupStore.New(s.db)
	permissionSetStorage := permissionSetStore.New(s.db)
	roleStorage := roleStore.New(s.db)

	// Load input dataset
	s.NoError(apiTokenStorage.UpsertMany(ctx, inputAPITokens))
	s.NoError(groupStorage.UpsertMany(ctx, inputGroups))
	s.NoError(permissionSetStorage.UpsertMany(ctx, inputPermissionSets))
	s.NoError(roleStorage.UpsertMany(ctx, inputRoles))

	// Run migration
	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
	}
	s.Require().NoError(migration.Run(dbs))

	// Fetch migrated dataset and validate storage content
	fetchedAPITokens := make([]*storage.TokenMetadata, 0)
	s.NoError(apiTokenStorage.Walk(ctx, func(obj *storage.TokenMetadata) error {
		fetchedAPITokens = append(fetchedAPITokens, obj)
		return nil
	}))
	s.ElementsMatch(expectedAPITokens, fetchedAPITokens)

	fetchedGroups := make([]*storage.Group, 0)
	s.NoError(groupStorage.Walk(ctx, func(obj *storage.Group) error {
		fetchedGroups = append(fetchedGroups, obj)
		return nil
	}))
	s.ElementsMatch(expectedGroups, fetchedGroups)

	fetchedPermissionSets := make([]*storage.PermissionSet, 0)
	s.NoError(permissionSetStorage.Walk(ctx, func(obj *storage.PermissionSet) error {
		fetchedPermissionSets = append(fetchedPermissionSets, obj)
		return nil
	}))
	s.ElementsMatch(expectedPermissionSets, fetchedPermissionSets)

	fetchedRoles := make([]*storage.Role, 0)
	s.NoError(roleStorage.Walk(ctx, func(obj *storage.Role) error {
		fetchedRoles = append(fetchedRoles, obj)
		return nil
	}))
	s.ElementsMatch(expectedRoles, fetchedRoles)
}
