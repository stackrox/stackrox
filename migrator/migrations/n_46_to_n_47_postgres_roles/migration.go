package n46ton47

// Code generation from pg-bindings generator disabled. To re-enable, check the gen.go file in
// central/role/store/role/postgres

import (
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	pkgMigrations "github.com/stackrox/rox/pkg/migrations"
	"gorm.io/gorm"
)

var (
	migration = types.Migration{
		StartingSeqNum: pkgMigrations.BasePostgresDBVersionSeqNum() + 46,
		VersionAfter:   &storage.Version{SeqNum: int32(pkgMigrations.BasePostgresDBVersionSeqNum()) + 47},
		Run: func(databases *types.Databases) error {
			// The data migration code was moved to the simpleaccessscope migrator.
			// The goal is to be able to convert the IDs that do not parse as UUIDs to proper UUID values,
			// and still be able to convert the references in the roles table.
			return nil
		},
	}
)

func move(gormDB *gorm.DB, postgresDB *pgxpool.Pool) error {
	return nil
}

func init() {
	migrations.MustRegisterMigration(migration)
}
