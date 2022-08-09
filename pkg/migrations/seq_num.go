package migrations

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/migrations/internal"
	"github.com/stackrox/rox/pkg/utils"
)

// CurrentDBVersionSeqNum is the current DB version number.
// This must be incremented every time we write a migration.
// It is a shared constant between central and the migrator binary.
func CurrentDBVersionSeqNum() int {
	// If PostgresDatastore is enabled, we consider all migrations to Postgres are following
	// the current migrations.
	// if features.PostgresDatastore.Enabled() {
	//	return internal.CurrentDBVersionSeqNum + internal.PostgresDBVersionPlus
	// }
	return internal.CurrentDBVersionSeqNum
}

// CurrentDBVersionSeqNumWithoutPostgres is the current DB version number
// without Postgres migrations. This function should only be used in testing
// environment.
func CurrentDBVersionSeqNumWithoutPostgres() int {
	if !features.PostgresDatastore.Enabled() {
		utils.Must(errors.New("Unexpected call, ROX_POSTGRES_DATASTORE is not true"))
	}
	return internal.CurrentDBVersionSeqNum
}
