//go:build sql_integration

package m197tom198

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/m_197_to_m_198_add_oidc_claim_mappings/schema"
	"github.com/stackrox/rox/migrator/migrations/m_197_to_m_198_add_oidc_claim_mappings/store/authproviders"
	"github.com/stackrox/rox/migrator/migrations/m_197_to_m_198_add_oidc_claim_mappings/store/groups"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

const (
	providerIDNamespace = "auth-provider-ns"
	groupIDNamespace    = "group-ns"
)

var ctx = sac.WithAllAccess(context.Background())

type apMigrationTestSuite struct {
	suite.Suite

	db *pghelper.TestPostgres
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(apMigrationTestSuite))
}

func (s *apMigrationTestSuite) SetupSuite() {
	s.db = pghelper.ForT(s.T(), false)
	pgutils.CreateTableFromModel(ctx, s.db.GetGormDB(), schema.CreateTableAuthProvidersStmt)
	pgutils.CreateTableFromModel(ctx, s.db.GetGormDB(), schema.CreateTableGroupsStmt)
}

func (s *apMigrationTestSuite) TearDownSuite() {
	s.db.Teardown(s.T())
}

var unmigratedAuthProviders = []*storage.AuthProvider{
	{
		Id:   uuid.NewV5FromNonUUIDs(providerIDNamespace, "no-claims").String(),
		Name: "no-claims",
		Type: "oidc",
	},
	{
		Id:   uuid.NewV5FromNonUUIDs(providerIDNamespace, "is-org-admin-group-declarative").String(),
		Name: "is-org-admin-group-declarative",
		Type: "oidc",
		Traits: &storage.Traits{
			Origin: storage.Traits_DECLARATIVE,
		},
	},
	{
		Id:   uuid.NewV5FromNonUUIDs(providerIDNamespace, "org-id-required-attribute").String(),
		Name: "org-id-required-attribute",
		Type: "oidc",
		RequiredAttributes: []*storage.AuthProvider_RequiredAttribute{
			{
				AttributeKey:   "orgid",
				AttributeValue: "anything",
			},
		},
	},
	{
		Id:   uuid.NewV5FromNonUUIDs(providerIDNamespace, "org-id-required-attribute-and-claim-mapping").String(),
		Name: "org-id-required-attribute-and-claim-mapping",
		Type: "oidc",
		RequiredAttributes: []*storage.AuthProvider_RequiredAttribute{
			{
				AttributeKey:   "orgid",
				AttributeValue: "anything",
			},
		},
		ClaimMappings: map[string]string{
			"account_id": "orgid",
		},
	},
	{
		Id:   uuid.NewV5FromNonUUIDs(providerIDNamespace, "org-id-group").String(),
		Name: "org-id-group",
		Type: "oidc",
	},
	{
		Id:   uuid.NewV5FromNonUUIDs(providerIDNamespace, "org-id-group-and-claim-mapping").String(),
		Name: "org-id-group-and-claim-mapping",
		Type: "oidc",
		ClaimMappings: map[string]string{
			"account_id": "orgid",
		},
	},
	{
		Id:   uuid.NewV5FromNonUUIDs(providerIDNamespace, "is-org-admin-required-attribute").String(),
		Name: "is-org-admin-required-attribute",
		Type: "oidc",
		RequiredAttributes: []*storage.AuthProvider_RequiredAttribute{
			{
				AttributeKey:   "groups",
				AttributeValue: "org_admin",
			},
		},
	},
	{
		Id:   uuid.NewV5FromNonUUIDs(providerIDNamespace, "is-org-admin-group").String(),
		Name: "is-org-admin-group",
		Type: "oidc",
	},
	{
		Id:   uuid.NewV5FromNonUUIDs(providerIDNamespace, "org-id-group-is-org-admin-required-attribute").String(),
		Name: "org-id-group-is-org-admin-required-attribute",
		Type: "oidc",
		RequiredAttributes: []*storage.AuthProvider_RequiredAttribute{
			{
				AttributeKey:   "groups",
				AttributeValue: "org_admin",
			},
		},
	},
}

var unmigratedGroups = []*storage.Group{
	{
		Props: &storage.GroupProperties{
			Id:             uuid.NewV5FromNonUUIDs(groupIDNamespace, "org-id-group").String(),
			AuthProviderId: uuid.NewV5FromNonUUIDs(providerIDNamespace, "org-id-group").String(),
			Key:            "orgid",
			Value:          "any",
		},
		RoleName: "Admin",
	},
	{
		Props: &storage.GroupProperties{
			Id:             uuid.NewV5FromNonUUIDs(groupIDNamespace, "is-org-admin-group-declarative").String(),
			AuthProviderId: uuid.NewV5FromNonUUIDs(providerIDNamespace, "is-org-admin-group-declarative").String(),
			Key:            "groups",
			Value:          "org_admin",
		},
		RoleName: "Admin",
	},
	{
		Props: &storage.GroupProperties{
			Id:             uuid.NewV5FromNonUUIDs(groupIDNamespace, "org-id-group-and-claim-mapping").String(),
			AuthProviderId: uuid.NewV5FromNonUUIDs(providerIDNamespace, "org-id-group-and-claim-mapping").String(),
			Key:            "orgid",
			Value:          "any",
		},
		RoleName: "Admin",
	},
	{
		Props: &storage.GroupProperties{
			Id:             uuid.NewV5FromNonUUIDs(groupIDNamespace, "is-org-admin-group").String(),
			AuthProviderId: uuid.NewV5FromNonUUIDs(providerIDNamespace, "is-org-admin-group").String(),
			Key:            "groups",
			Value:          "org_admin",
		},
		RoleName: "Admin",
	},
	{
		Props: &storage.GroupProperties{
			Id:             uuid.NewV5FromNonUUIDs(groupIDNamespace, "org-id-group-is-org-admin-required-attribute").String(),
			AuthProviderId: uuid.NewV5FromNonUUIDs(providerIDNamespace, "org-id-group-is-org-admin-required-attribute").String(),
			Key:            "orgid",
			Value:          "any",
		},
		RoleName: "Admin",
	},
}

var migratedAuthProviders = []*storage.AuthProvider{
	{
		Id:   uuid.NewV5FromNonUUIDs(providerIDNamespace, "no-claims").String(),
		Name: "no-claims",
		Type: "oidc",
	},
	{
		Id:   uuid.NewV5FromNonUUIDs(providerIDNamespace, "org-id-required-attribute").String(),
		Name: "org-id-required-attribute",
		Type: "oidc",
		RequiredAttributes: []*storage.AuthProvider_RequiredAttribute{
			{
				AttributeKey:   "orgid",
				AttributeValue: "anything",
			},
		},
		ClaimMappings: map[string]string{
			"account_id": "orgid",
		},
	},
	{
		Id:   uuid.NewV5FromNonUUIDs(providerIDNamespace, "is-org-admin-group-declarative").String(),
		Name: "is-org-admin-group-declarative",
		Type: "oidc",
		Traits: &storage.Traits{
			Origin: storage.Traits_DECLARATIVE,
		},
	},
	{
		Id:   uuid.NewV5FromNonUUIDs(providerIDNamespace, "org-id-required-attribute-and-claim-mapping").String(),
		Name: "org-id-required-attribute-and-claim-mapping",
		Type: "oidc",
		RequiredAttributes: []*storage.AuthProvider_RequiredAttribute{
			{
				AttributeKey:   "orgid",
				AttributeValue: "anything",
			},
		},
		ClaimMappings: map[string]string{
			"account_id": "orgid",
		},
	},
	{
		Id:   uuid.NewV5FromNonUUIDs(providerIDNamespace, "org-id-group").String(),
		Name: "org-id-group",
		Type: "oidc",
		ClaimMappings: map[string]string{
			"account_id": "orgid",
		},
	},
	{
		Id:   uuid.NewV5FromNonUUIDs(providerIDNamespace, "org-id-group-and-claim-mapping").String(),
		Name: "org-id-group-and-claim-mapping",
		Type: "oidc",
		ClaimMappings: map[string]string{
			"account_id": "orgid",
		},
	},
	{
		Id:   uuid.NewV5FromNonUUIDs(providerIDNamespace, "is-org-admin-required-attribute").String(),
		Name: "is-org-admin-required-attribute",
		Type: "oidc",
		RequiredAttributes: []*storage.AuthProvider_RequiredAttribute{
			{
				AttributeKey:   "org_admin",
				AttributeValue: "true",
			},
		},
		ClaimMappings: map[string]string{
			"is_org_admin": "org_admin",
		},
	},
	{
		Id:   uuid.NewV5FromNonUUIDs(providerIDNamespace, "is-org-admin-group").String(),
		Name: "is-org-admin-group",
		Type: "oidc",
		ClaimMappings: map[string]string{
			"is_org_admin": "org_admin",
		},
	},
	{
		Id:   uuid.NewV5FromNonUUIDs(providerIDNamespace, "org-id-group-is-org-admin-required-attribute").String(),
		Name: "org-id-group-is-org-admin-required-attribute",
		Type: "oidc",
		RequiredAttributes: []*storage.AuthProvider_RequiredAttribute{
			{
				AttributeKey:   "org_admin",
				AttributeValue: "true",
			},
		},
		ClaimMappings: map[string]string{
			"account_id":   "orgid",
			"is_org_admin": "org_admin",
		},
	},
}

var migratedGroups = []*storage.Group{
	{
		Props: &storage.GroupProperties{
			Id:             uuid.NewV5FromNonUUIDs(groupIDNamespace, "org-id-group").String(),
			AuthProviderId: uuid.NewV5FromNonUUIDs(providerIDNamespace, "org-id-group").String(),
			Key:            "orgid",
			Value:          "any",
		},
		RoleName: "Admin",
	},
	{
		Props: &storage.GroupProperties{
			Id:             uuid.NewV5FromNonUUIDs(groupIDNamespace, "is-org-admin-group-declarative").String(),
			AuthProviderId: uuid.NewV5FromNonUUIDs(providerIDNamespace, "is-org-admin-group-declarative").String(),
			Key:            "groups",
			Value:          "org_admin",
		},
		RoleName: "Admin",
	},
	{
		Props: &storage.GroupProperties{
			Id:             uuid.NewV5FromNonUUIDs(groupIDNamespace, "org-id-group-and-claim-mapping").String(),
			AuthProviderId: uuid.NewV5FromNonUUIDs(providerIDNamespace, "org-id-group-and-claim-mapping").String(),
			Key:            "orgid",
			Value:          "any",
		},
		RoleName: "Admin",
	},
	{
		Props: &storage.GroupProperties{
			Id:             uuid.NewV5FromNonUUIDs(groupIDNamespace, "is-org-admin-group").String(),
			AuthProviderId: uuid.NewV5FromNonUUIDs(providerIDNamespace, "is-org-admin-group").String(),
			Key:            "org_admin",
			Value:          "true",
		},
		RoleName: "Admin",
	},
	{
		Props: &storage.GroupProperties{
			Id:             uuid.NewV5FromNonUUIDs(groupIDNamespace, "org-id-group-is-org-admin-required-attribute").String(),
			AuthProviderId: uuid.NewV5FromNonUUIDs(providerIDNamespace, "org-id-group-is-org-admin-required-attribute").String(),
			Key:            "orgid",
			Value:          "any",
		},
		RoleName: "Admin",
	},
}

func (s *apMigrationTestSuite) TestMigration() {
	groupStore := groups.New(s.db.DB)
	providerStore := authproviders.New(s.db.DB)

	s.Require().NoError(providerStore.UpsertMany(ctx, unmigratedAuthProviders))
	s.Require().NoError(groupStore.UpsertMany(ctx, unmigratedGroups))

	dbs := &types.Databases{
		PostgresDB: s.db.DB,
	}

	s.Require().NoError(migration.Run(dbs))

	allAPAfterMigration := make([]*storage.AuthProvider, 0, len(unmigratedAuthProviders))
	s.NoError(providerStore.Walk(ctx, func(obj *storage.AuthProvider) error {
		allAPAfterMigration = append(allAPAfterMigration, obj)
		return nil
	}))

	s.ElementsMatch(migratedAuthProviders, allAPAfterMigration)

	allGroupsAfterMigration := make([]*storage.Group, 0, len(unmigratedGroups))
	s.NoError(groupStore.Walk(ctx, func(obj *storage.Group) error {
		allGroupsAfterMigration = append(allGroupsAfterMigration, obj)
		return nil
	}))

	s.ElementsMatch(migratedGroups, allGroupsAfterMigration)
}
