package migrations

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/migrations/internal"
	"github.com/stackrox/rox/pkg/utils"
)

// CurrentDBVersionSeqNum is the current DB version number.
// This must be incremented every time we write a migration.
// It is a shared constant between central and the migrator binary.
func CurrentDBVersionSeqNum() int {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return internal.CurrentDBVersionSeqNum
	}
	// XXX: to remove. The following is just to test how many test would fail
	utils.Should(errors.Errorf("ROX_POSTGRES_DATASTORE should be true. We do not support RocksDB anymore."))
	return internal.LastRocksDBVersionSeqNum
}

// BasePostgresDBVersionSeqNum is the base of DB version number
// for Postgres migrations. This function should only be used in Postgres
// migrations.
func BasePostgresDBVersionSeqNum() int {
	return internal.LastRocksDBVersionSeqNum - 1
}
