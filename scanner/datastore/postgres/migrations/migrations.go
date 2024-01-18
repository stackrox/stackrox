package migrations

import (
	"database/sql"
	"embed"

	"github.com/remind101/migrate"
)

// MatcherMigrationTable specifies the name of the matcher migration table.
const MatcherMigrationTable = "matcher_migrations"

// MatcherMigrations lists the matcher migrations, in order.
var MatcherMigrations = []migrate.Migration{
	{
		ID: 1,
		Up: runFile("matcher/01-init.sql"),
	},
}

//go:embed */*.sql
var fs embed.FS

func runFile(n string) func(*sql.Tx) error {
	b, err := fs.ReadFile(n)
	return func(tx *sql.Tx) error {
		if err != nil {
			return err
		}
		if _, err := tx.Exec(string(b)); err != nil {
			return err
		}
		return nil
	}
}
