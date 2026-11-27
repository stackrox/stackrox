//go:build sql_integration

package m230tom231

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	oldSchema "github.com/stackrox/rox/migrator/migrations/m_230_to_m_231_drop_machine_to_machine_configuration_unique_index_per_issuer/schema/old"
	newM2MStore "github.com/stackrox/rox/migrator/migrations/m_230_to_m_231_drop_machine_to_machine_configuration_unique_index_per_issuer/store/new/auth_machine_to_machine_configs"
	oldM2MStore "github.com/stackrox/rox/migrator/migrations/m_230_to_m_231_drop_machine_to_machine_configuration_unique_index_per_issuer/store/old/auth_machine_to_machine_configs"
	oldRoleStore "github.com/stackrox/rox/migrator/migrations/m_230_to_m_231_drop_machine_to_machine_configuration_unique_index_per_issuer/store/old/roles"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
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
	s.ctx = sac.WithAllAccess(s.T().Context())
	s.db = pghelper.ForT(s.T(), false)

	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), oldSchema.CreateTableRolesStmt)
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), oldSchema.CreateTableAuthMachineToMachineConfigsStmt)
}

func getM2MConfig(issuer string, userEMail string) *storage.AuthMachineToMachineConfig {
	return &storage.AuthMachineToMachineConfig{
		Id:                      uuid.NewV4().String(),
		Type:                    storage.AuthMachineToMachineConfig_GENERIC,
		TokenExpirationDuration: "1h",
		Mappings: []*storage.AuthMachineToMachineConfig_Mapping{
			{
				Key:             "email",
				ValueExpression: userEMail,
				Role:            "Analyst",
			},
		},
		Issuer: issuer,
	}
}

func (s *migrationTestSuite) TestMigration() {
	m2mOldStore := oldM2MStore.New(s.db)
	roleOldStore := oldRoleStore.New(s.db)

	const m2mIssuer = "test-issuer"
	analystRole := &storage.Role{
		Name:            "Analyst",
		PermissionSetId: "ffffffff-ffff-fff4-f5ff-fffffffffffe", // Analyst
		AccessScopeId:   "ffffffff-ffff-fff4-f5ff-ffffffffffff", // unrestricted
	}
	m2mConfig1 := getM2MConfig(m2mIssuer, "my-user@example.com")
	m2mConfig2 := getM2MConfig(m2mIssuer, "my-other-user@example.com")
	m2mConfig3 := getM2MConfig(m2mIssuer, "yet-another-user@example.com")

	err := roleOldStore.Upsert(s.ctx, analystRole)
	assert.NoError(s.T(), err)
	store1Err := m2mOldStore.Upsert(s.ctx, m2mConfig1)
	assert.NoError(s.T(), store1Err)

	// Ensure the unique constraint prevents writing the second or third config for the m2mIssuer
	store2Err := m2mOldStore.Upsert(s.ctx, m2mConfig2)
	assert.ErrorContains(s.T(), store2Err, `violates unique constraint "uni_auth_machine_to_machine_configs_issuer"`)
	store3Err := m2mOldStore.Upsert(s.ctx, m2mConfig3)
	assert.ErrorContains(s.T(), store3Err, `violates unique constraint "uni_auth_machine_to_machine_configs_issuer"`)

	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}

	s.Require().NoError(migration.Run(dbs))

	m2mNewStore := newM2MStore.New(s.db)

	fetched := make([]*storage.AuthMachineToMachineConfig, 0, 1)
	walk1Err := m2mNewStore.Walk(s.ctx, func(obj *storage.AuthMachineToMachineConfig) error {
		fetched = append(fetched, obj)
		return nil
	})
	assert.NoError(s.T(), walk1Err)
	protoassert.ElementsMatch(s.T(), []*storage.AuthMachineToMachineConfig{m2mConfig1}, fetched)

	// Validate that the unique issuer constraint is not there (enforced) anymore,
	// and that both old and new versions of the store can write to the table.
	store2AgainErr := m2mNewStore.Upsert(s.ctx, m2mConfig2)
	assert.NoError(s.T(), store2AgainErr)
	store3AgainErr := m2mOldStore.Upsert(s.ctx, m2mConfig3)
	assert.NoError(s.T(), store3AgainErr)

	fetched2 := make([]*storage.AuthMachineToMachineConfig, 0, 3)
	walk2Err := m2mNewStore.Walk(s.ctx, func(obj *storage.AuthMachineToMachineConfig) error {
		fetched2 = append(fetched2, obj)
		return nil
	})
	assert.NoError(s.T(), walk2Err)
	protoassert.ElementsMatch(s.T(), []*storage.AuthMachineToMachineConfig{m2mConfig1, m2mConfig2, m2mConfig3}, fetched2)

	fetched3 := make([]*storage.AuthMachineToMachineConfig, 0, 3)
	walk3Err := m2mNewStore.Walk(s.ctx, func(obj *storage.AuthMachineToMachineConfig) error {
		fetched3 = append(fetched3, obj)
		return nil
	})
	assert.NoError(s.T(), walk3Err)
	protoassert.ElementsMatch(s.T(), []*storage.AuthMachineToMachineConfig{m2mConfig3, m2mConfig2, m2mConfig1}, fetched3)
}
