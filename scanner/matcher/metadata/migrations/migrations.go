package migrations

import (
	"database/sql"
	"embed"

	"github.com/remind101/migrate"
)

const MigrationTable = "last_vuln_update_migrations"

var Migrations = []migrate.Migration{
	{
		ID: 1,
		Up: runFile("01-init.sql"),
	},
}

//go:embed *.sql
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
