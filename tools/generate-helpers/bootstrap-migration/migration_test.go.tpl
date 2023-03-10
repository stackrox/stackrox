package {{.packageName}}

import (
    "testing"

    pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stretchr/testify/assert"
)

func TestMigration(t *testing.T) {
    postgres := pghelper.ForT(t, true)
    defer postgres.TearDown(t)

    databases := &types.Databases{
    	GormDB:     s.db.GetGormDB(),
    	PostgresDB: s.db.DB,
    }

    // TODO: populate database prior to migration

    assert.NoError(migration.Run(dbs))

    // TODO: validate database content post migration
}
