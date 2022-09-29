package migrations

import (
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/migrations/internal"
)

// CurrentDBVersionSeqNum is the current DB version number.
// This must be incremented every time we write a migration.
// It is a shared constant between central and the migrator binary.
func CurrentDBVersionSeqNum() int {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return internal.CurrentDBVersionSeqNum + internal.PostgresDBVersionPlus
	}
	return internal.CurrentDBVersionSeqNum
}

// CurrentDBVersionSeqNumWithoutPostgres is the base of current DB version number
// without Postgres migrations. This function should only be used in Postgres
// migrations.
func CurrentDBVersionSeqNumWithoutPostgres() int {
	return internal.CurrentDBVersionSeqNum - 1
}
