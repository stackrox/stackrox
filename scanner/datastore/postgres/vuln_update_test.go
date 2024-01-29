//go:build scanner_db_integration

package postgres

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVulnUpdateStore(t *testing.T) {
	ctx := context.Background()
	pool := testDB(t, ctx, "vuln_update_test")
	store, err := InitPostgresMatcherMetadataStore(ctx, pool, true)
	require.NoError(t, err)

	// Initial timestamp should be "empty"
	timestamp, err := store.GetLastVulnerabilityUpdate(ctx)
	require.NoError(t, err)
	assert.Equal(t, time.Time{}, timestamp)

	now, err := http.ParseTime(time.Now().UTC().Format(http.TimeFormat))
	require.NoError(t, err)

	err = store.SetLastVulnerabilityUpdate(ctx, now)
	require.NoError(t, err)
	timestamp, err = store.GetLastVulnerabilityUpdate(ctx)
	require.NoError(t, err)
	assert.Equal(t, now, timestamp)
}
