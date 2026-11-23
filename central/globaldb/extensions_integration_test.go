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

	// The installed version matches the server's default version, so this is a
	// no-op.
	require.NoError(t, updateExtensionIfOutdated(ctx, tp.DB, "pg_stat_statements"))

	var version string
	require.NoError(t, tp.DB.QueryRow(ctx,
		"SELECT extversion FROM pg_extension WHERE extname = 'pg_stat_statements'").Scan(&version))
	require.NotEmpty(t, version)

	// An extension that is not installed is left alone.
	require.NoError(t, updateExtensionIfOutdated(ctx, tp.DB, "extension_that_does_not_exist"))
}
