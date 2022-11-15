package n37ton38

// Code generation from pg-bindings generator disabled. To re-enable, check the gen.go file in
// central/role/store/permissionset/postgres

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	pkgMigrations "github.com/stackrox/rox/pkg/migrations"
)

var (
	migration = types.Migration{
		StartingSeqNum: pkgMigrations.CurrentDBVersionSeqNumWithoutPostgres() + 37,
		VersionAfter:   &storage.Version{SeqNum: int32(pkgMigrations.CurrentDBVersionSeqNumWithoutPostgres()) + 38},
		Run: func(databases *types.Databases) error {
			// The data migration code was moved to the simpleaccessscope migrator.
			// The goal is to be able to convert the IDs that do not parse as UUIDs to proper UUID values,
			// and still be able to convert the references in the roles table.
			return nil
		},
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}
