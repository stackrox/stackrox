package migrations

import (
	"database/sql"
	"embed"

	"github.com/remind101/migrate"
	"github.com/stackrox/rox/pkg/features"
)

const (
	// IndexerMigrationTable specifies the name of the indexer migration table.
	IndexerMigrationTable = "indexer_migrations"

	// MatcherMigrationTable specifies the name of the matcher migration table.
	MatcherMigrationTable = "matcher_migrations"
)

// IndexerMigrations lists the indexer migrations (not necessarily in order).
var IndexerMigrations = []migrate.Migration{
	{
		ID: 2,
		Up: runFile("indexer/02-external-index-report.sql"),
	},
}

func init() {
	if features.ScannerV4ReIndex.Enabled() {
		IndexerMigrations = append(IndexerMigrations, migrate.Migration{
			ID: 1,
			Up: runFile("indexer/01-init.sql"),
		})
	}
}

// MatcherMigrations lists the matcher migrations, in order.
var MatcherMigrations = []migrate.Migration{
	{
		ID: 1,
		Up: runFile("matcher/01-init.sql"),
	},
	{
		ID:   2,
		Up:   runFile("matcher/02-update-timestamp-column-last-vuln-update-up.sql"),
		Down: runFile("matcher/02-update-timestamp-column-last-vuln-update-down.sql"),
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
