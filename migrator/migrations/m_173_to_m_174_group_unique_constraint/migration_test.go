package m173tom174

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	frozenSchemav73 "github.com/stackrox/rox/migrator/migrations/frozenschema/v73"
	"github.com/stackrox/rox/migrator/migrations/m_173_to_m_174_group_unique_constraint/groupspostgresstore"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

var (
	ctx = sac.WithAllAccess(context.Background())

	groupsPreMigration = []*storage.Group{
		{
			Props: &storage.GroupProperties{
				Id:             "group-id-1",
				AuthProviderId: "auth-provider-1",
				Key:            "",
				Value:          "",
			},
			RoleName: "Admin",
		},
		{
			Props: &storage.GroupProperties{
				Id:             "group-id-2",
				AuthProviderId: "auth-provider-1",
				Key:            "email",
				Value:          "someone@example.com",
			},
			RoleName: "Analyst",
		},
		{
			Props: &storage.GroupProperties{
				Id:             "group-id-3",
				AuthProviderId: "auth-provider-1",
				Key:            "email",
				Value:          "someone@example.com",
			},
			RoleName: "Admin",
		},
		{
			Props: &storage.GroupProperties{
				Id:             "group-id-4",
				AuthProviderId: "auth-provider-1",
				Key:            "email",
				Value:          "someone@example.com",
			},
			RoleName: "Admin",
		},
		{
			Props: &storage.GroupProperties{
				Id:             "group-id-5",
				AuthProviderId: "auth-provider-2",
				Key:            "email",
				Value:          "someone@example.com",
			},
			RoleName: "Admin",
		},
	}

	groupsPostMigration = []*storage.Group{
		{
			Props: &storage.GroupProperties{
				Id:             "group-id-1",
				AuthProviderId: "auth-provider-1",
				Key:            "",
				Value:          "",
			},
			RoleName: "Admin",
		},
		{
			Props: &storage.GroupProperties{
				Id:             "group-id-2",
				AuthProviderId: "auth-provider-1",
				Key:            "email",
				Value:          "someone@example.com",
			},
			RoleName: "Analyst",
		},
		{
			Props: &storage.GroupProperties{
				Id:             "group-id-3",
				AuthProviderId: "auth-provider-1",
				Key:            "email",
				Value:          "someone@example.com",
			},
			RoleName: "Admin",
		},
		{
			Props: &storage.GroupProperties{
				Id:             "group-id-5",
				AuthProviderId: "auth-provider-2",
				Key:            "email",
				Value:          "someone@example.com",
			},
			RoleName: "Admin",
		},
	}
)

type groupUniqueConstraintMigrationTestSuite struct {
	suite.Suite

	db *pghelper.TestPostgres
}

func TestGroupUniqueConstraintMigration(t *testing.T) {
	suite.Run(t, new(groupUniqueConstraintMigrationTestSuite))
}

func (s *groupUniqueConstraintMigrationTestSuite) SetupSuite() {
	s.db = pghelper.ForT(s.T(), true)
	pgutils.CreateTableFromModel(ctx, s.db.GetGormDB(), frozenSchemav73.CreateTableGroupsStmt)
}

func (s *groupUniqueConstraintMigrationTestSuite) TearDownSuite() {
	s.db.Teardown(s.T())
}

func (s *groupUniqueConstraintMigrationTestSuite) TestMigration() {
	store := groupspostgresstore.New(s.db.DB)

	s.Require().NoError(store.UpsertMany(ctx, groupsPreMigration))

	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
	}

	s.Require().NoError(migration.Run(dbs))

	groupsAfterMigration := make([]*storage.Group, 0, len(groupsPreMigration))

	s.NoError(store.Walk(ctx, func(group *storage.Group) error {
		groupsAfterMigration = append(groupsAfterMigration, group)
		return nil
	}))

	s.ElementsMatch(groupsPostMigration, groupsAfterMigration)
}
