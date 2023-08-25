//go:build sql_integration

package m173tom174

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	frozenSchemav73 "github.com/stackrox/rox/migrator/migrations/frozenschema/v73"
	"github.com/stackrox/rox/migrator/migrations/m_173_to_m_174_group_unique_constraint/stores/previous"
	"github.com/stackrox/rox/migrator/migrations/m_173_to_m_174_group_unique_constraint/stores/updated"
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

	groupsPostMigration = map[string]*storage.Group{
		"group-id-1": {
			Props: &storage.GroupProperties{
				Id:             "group-id-1",
				AuthProviderId: "auth-provider-1",
				Key:            "",
				Value:          "",
			},
			RoleName: "Admin",
		},
		"group-id-2": {
			Props: &storage.GroupProperties{
				Id:             "group-id-2",
				AuthProviderId: "auth-provider-1",
				Key:            "email",
				Value:          "someone@example.com",
			},
			RoleName: "Analyst",
		},
		"group-id-5": {
			Props: &storage.GroupProperties{
				Id:             "group-id-5",
				AuthProviderId: "auth-provider-2",
				Key:            "email",
				Value:          "someone@example.com",
			},
			RoleName: "Admin",
		},
	}

	prunedGroupPostMigration = &storage.Group{
		Props: &storage.GroupProperties{
			AuthProviderId: "auth-provider-1",
			Key:            "email",
			Value:          "someone@example.com",
		},
		RoleName: "Admin",
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
	s.db = pghelper.ForT(s.T(), false)
	pgutils.CreateTableFromModel(ctx, s.db.GetGormDB(), frozenSchemav73.CreateTableGroupsStmt)
}

func (s *groupUniqueConstraintMigrationTestSuite) TearDownSuite() {
	s.db.Teardown(s.T())
}

func (s *groupUniqueConstraintMigrationTestSuite) TestMigration() {
	previousStore := previous.New(s.db.DB)
	updatedStore := updated.New(s.db.DB)

	s.Require().NoError(previousStore.UpsertMany(ctx, groupsPreMigration))

	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
	}

	s.Require().NoError(migration.Run(dbs))

	groupsAfterMigration := make([]*storage.Group, 0, len(groupsPreMigration))

	s.NoError(updatedStore.Walk(ctx, func(group *storage.Group) error {
		groupsAfterMigration = append(groupsAfterMigration, group)
		return nil
	}))

	s.Len(groupsAfterMigration, len(groupsPostMigration)+1)

	var prunedGroupFound bool
	for _, group := range groupsAfterMigration {
		if expectedGroup, exists := groupsPostMigration[group.GetProps().GetId()]; exists {
			s.Equal(expectedGroup, group)
		} else {
			s.False(prunedGroupFound, "found the pruned group twice")
			prunedGroupFound = true
			s.Equal(prunedGroupPostMigration.GetRoleName(), group.GetRoleName())
			s.Equal(prunedGroupPostMigration.GetProps().GetAuthProviderId(), group.GetProps().GetAuthProviderId())
			s.Equal(prunedGroupPostMigration.GetProps().GetKey(), group.GetProps().GetKey())
			s.Equal(prunedGroupPostMigration.GetProps().GetValue(), group.GetProps().GetValue())
		}
	}
}
