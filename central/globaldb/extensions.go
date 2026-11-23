package globaldb

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/stackrox/rox/pkg/postgres"
)

// updateExtensionIfOutdated updates an installed extension when the version
// shipped by the running PostgreSQL server differs from the installed one.
// pg_upgrade copies an extension's schema but does not update its version, and
// CREATE EXTENSION IF NOT EXISTS is a no-op for an existing older version, so
// an outdated extension can otherwise persist after a major-version upgrade.
//
// The update is per-database; call it on a connection to the active database.
// Extensions that are not installed are left alone, and a matching version is
// a no-op.
func updateExtensionIfOutdated(ctx context.Context, db postgres.DB, name string) error {
	var installedVersion, availableVersion string
	err := db.QueryRow(ctx, `
		SELECT e.extversion, a.default_version
		FROM pg_extension e
		JOIN pg_available_extensions a ON a.name = e.extname
		WHERE e.extname = $1`, name).Scan(&installedVersion, &availableVersion)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("reading %q extension version: %w", name, err)
	}
	if !needsExtensionUpdate(installedVersion, availableVersion) {
		return nil
	}
	if _, err := db.Exec(ctx, "ALTER EXTENSION "+pgx.Identifier{name}.Sanitize()+" UPDATE"); err != nil {
		return fmt.Errorf("updating %q extension from %s to %s: %w", name, installedVersion, availableVersion, err)
	}
	return nil
}

// needsExtensionUpdate reports whether the installed extension version differs
// from the version shipped by the server. Only equality matters: PostgreSQL
// version strings are not ordered lexicographically (for example, "1.10" sorts
// before "1.9"), and ALTER EXTENSION ... UPDATE is a no-op when they match.
func needsExtensionUpdate(installedVersion, availableVersion string) bool {
	return installedVersion != availableVersion
}
