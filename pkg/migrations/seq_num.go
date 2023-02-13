package migrations

import (
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/mathutil"
	"github.com/stackrox/rox/pkg/migrations/internal"
)

// CurrentDBVersionSeqNum is the current DB version number.
// This must be incremented every time we write a migration.
// It is a shared constant between central and the migrator binary.
func CurrentDBVersionSeqNum() int {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return internal.CurrentDBVersionSeqNum
	}
	return mathutil.MinInt(internal.LastRocksDBVersionSeqNum, internal.CurrentDBVersionSeqNum)
}

// BasePostgresDBVersionSeqNum is the base of DB version number
// for Postgres migrations. This function should only be used in Postgres
// migrations.
func BasePostgresDBVersionSeqNum() int {
	return internal.LastRocksDBVersionSeqNum - 1
}

// LastRocksDBVersionSeqNum is the sequence number for the last RocksDB version.
func LastRocksDBVersionSeqNum() int {
	return internal.LastRocksDBVersionSeqNum
}

// LastRocksToPostgresDBVersionSeqNum is the sequence number for the last RocksDB to Postgres version.
func LastRocksToPostgresDBVersionSeqNum() int {
	return internal.LastRocksToPostgresDBVersionSeqNum
}
