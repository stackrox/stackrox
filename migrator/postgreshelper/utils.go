package postgreshelper

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgadmin"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/utils"
)

// RenameDB - renames a database
func RenameDB(adminPool postgres.DB, originalDB, newDB string) error {
	if pgconfig.IsExternalDatabase() {
		utils.Should(errors.New("unexpected call, should not try to rename a database from an external Postgres."))
	}

	log.WriteToStderrf("Renaming database %q to %q", originalDB, newDB)
	ctx, cancel := context.WithTimeout(context.Background(), pgadmin.PostgresQueryTimeout)
	defer cancel()

	// Terminate connections to the source database.  You cannot move a database if
	// there are open connections to it.
	err := pgadmin.TerminateConnection(adminPool.Config(), originalDB)
	if err != nil {
		return err
	}

	sqlStmt := fmt.Sprintf("ALTER DATABASE %s RENAME TO %s", originalDB, newDB)

	_, err = adminPool.Exec(ctx, sqlStmt)

	return err
}
