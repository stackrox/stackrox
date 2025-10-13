//go:build sql_integration

package m205tom206

import (
	"context"
	"testing"

	oldSchema "github.com/stackrox/rox/migrator/migrations/m_205_to_m_206_remove_bad_gorm_index/schema/old"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

const (
	constraintQuery = `SELECT conname FROM pg_constraint WHERE conrelid = (SELECT oid FROM pg_class WHERE relname LIKE 'compliance_integrations' and conname = 'idx_compliance_integrations_clusterid')`
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

	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), oldSchema.CreateTableComplianceIntegrationsStmt)
}

func (s *migrationTestSuite) TestMigration() {
	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}

	// Have to add the bad constraint manually as GORM is who put it in there and that version
	// of GORM is long gone from the repo.
	tx := dbs.GormDB.Exec("ALTER TABLE compliance_integrations ADD CONSTRAINT idx_compliance_integrations_clusterid UNIQUE (clusterid);")
	s.Require().NoError(tx.Error)
	// Verify the constraint is there.
	s.Require().True(s.badConstraintExists())

	// Verify that applying the schema fails
	err := dbs.GormDB.WithContext(s.ctx).AutoMigrate(oldSchema.CreateTableComplianceIntegrationsStmt.GormModel)
	// I wanted to make this required, but I was concerned GORM would actually fix the issue and then
	// this would succeed.
	if err != nil {
		log.Errorf("error creating table compliance integrations: %v", err)
	}

	// Run the migration
	s.Require().NoError(migration.Run(dbs))
	// Verify the constraint is not there
	s.Require().False(s.badConstraintExists())

	// Run it again now that constraint is gone
	s.Require().NoError(migration.Run(dbs))

	// Verify that applying the schema succeeds
	err = dbs.GormDB.WithContext(s.ctx).AutoMigrate(oldSchema.CreateTableComplianceIntegrationsStmt.GormModel)
	s.Require().NoError(err)
	// Verify the constraint is not there
	s.Require().False(s.badConstraintExists())
}

func (s *migrationTestSuite) badConstraintExists() bool {
	tx := s.db.GetGormDB().Exec(constraintQuery)
	s.Require().NoError(tx.Error)

	return tx.RowsAffected == 1
}
