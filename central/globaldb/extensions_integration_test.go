//go:build sql_integration

package globaldb

import (
	"context"
	"testing"

	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stretchr/testify/require"
)

func TestUpdateExtensionIfOutdated(t *testing.T) {
	ctx := context.Background()
	tp := pgtest.ForT(t)

	_, err := tp.DB.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS pg_stat_statements")
	require.NoError(t, err)

	var defaultVersion string
	require.NoError(t, tp.DB.QueryRow(ctx,
		"SELECT default_version FROM pg_available_extensions WHERE name = 'pg_stat_statements'").Scan(&defaultVersion))
	require.NotEmpty(t, defaultVersion)

	installedVersion := func() string {
		var version string
		require.NoError(t, tp.DB.QueryRow(ctx,
			"SELECT extversion FROM pg_extension WHERE extname = 'pg_stat_statements'").Scan(&version))
		return version
	}

	// The installed version matches the server's default version, so this is a
	// no-op.
	require.NoError(t, updateExtensionIfOutdated(ctx, tp.DB, "pg_stat_statements"))
	require.Equal(t, defaultVersion, installedVersion())

	// Force the recorded version behind the server's to exercise the update
	// path. The extension ships forward update scripts, so ALTER EXTENSION ...
	// UPDATE restores the server's default version.
	_, err = tp.DB.Exec(ctx,
		"UPDATE pg_extension SET extversion = '1.9' WHERE extname = 'pg_stat_statements'")
	require.NoError(t, err)
	require.NoError(t, updateExtensionIfOutdated(ctx, tp.DB, "pg_stat_statements"))
	require.Equal(t, defaultVersion, installedVersion())

	// An extension that is not installed is left alone.
	require.NoError(t, updateExtensionIfOutdated(ctx, tp.DB, "extension_that_does_not_exist"))
}

func TestUpdateExtensionIfOutdatedConcurrent(t *testing.T) {
	ctx := context.Background()
	tp := pgtest.ForT(t)

	_, err := tp.DB.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS pg_stat_statements")
	require.NoError(t, err)

	var defaultVersion string
	require.NoError(t, tp.DB.QueryRow(ctx,
		"SELECT default_version FROM pg_available_extensions WHERE name = 'pg_stat_statements'").Scan(&defaultVersion))

	// Force the recorded version behind the server's so concurrent callers race
	// to apply the same update, as multiple Central replicas do at startup.
	_, err = tp.DB.Exec(ctx,
		"UPDATE pg_extension SET extversion = '1.9' WHERE extname = 'pg_stat_statements'")
	require.NoError(t, err)

	const workers = 4
	errCh := make(chan error, workers)
	for i := 0; i < workers; i++ {
		go func() {
			errCh <- updateExtensionIfOutdated(ctx, tp.DB, "pg_stat_statements")
		}()
	}
	for i := 0; i < workers; i++ {
		require.NoError(t, <-errCh)
	}

	var version string
	require.NoError(t, tp.DB.QueryRow(ctx,
		"SELECT extversion FROM pg_extension WHERE extname = 'pg_stat_statements'").Scan(&version))
	require.Equal(t, defaultVersion, version)
}
