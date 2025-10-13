//go:build scanner_db_integration

package postgres

import (
	"context"
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

	now := time.Now()
	later := now.Add(time.Hour)

	err = store.SetLastVulnerabilityUpdate(ctx, "now", now)
	require.NoError(t, err)
	err = store.SetLastVulnerabilityUpdate(ctx, "later", later)
	require.NoError(t, err)

	timestamp, err = store.GetLastVulnerabilityUpdate(ctx)
	require.NoError(t, err)
	assert.Equal(t, now.UTC().Truncate(time.Second), timestamp, "now does not match")

	// Get or init a new key, verify the update time is older.
	newT := timestamp.Add(-24 * time.Hour)
	timestamp, err = store.GetOrSetLastVulnerabilityUpdate(ctx, "new", newT)
	require.NoError(t, err)
	assert.Equal(t, newT, timestamp, "new was set vuln update time")
	timestamp, err = store.GetLastVulnerabilityUpdate(ctx)
	require.NoError(t, err)
	assert.Equal(t, newT, timestamp, "new did not change the vuln update time")
}

func Test_CleanVulnerabilityUpdates(t *testing.T) {
	ctx := context.Background()
	pool := testDB(t, ctx, "vuln_update_test")
	store, err := InitPostgresMatcherMetadataStore(ctx, pool, true)
	require.NoError(t, err)
	now := time.Now()

	// Insert older invalid entry (expected to be cleaned-up), and valid and a future
	// (expected to not be cleaned-up).
	err = store.SetLastVulnerabilityUpdate(ctx, "invalid", now.Add(-2*time.Hour))
	require.NoError(t, err)

	err = store.SetLastVulnerabilityUpdate(ctx, "valid", now.Add(-time.Hour))
	require.NoError(t, err)

	err = store.SetLastVulnerabilityUpdate(ctx, "new", now)
	require.NoError(t, err)

	// Verify invalid timestamp is returned when not cleaned.
	timestamp, err := store.GetLastVulnerabilityUpdate(ctx)
	require.NoError(t, err)
	assert.Equal(t, now.Add(-2*time.Hour).UTC().Truncate(time.Second), timestamp)

	err = store.GCVulnerabilityUpdates(ctx, []string{"valid"}, now)
	require.NoError(t, err)

	// Expect valid timestamp after cleaned.
	timestamp, err = store.GetLastVulnerabilityUpdate(ctx)
	require.NoError(t, err)
	assert.Equal(t, now.Add(-time.Hour).UTC().Truncate(time.Second), timestamp)

	// Update valid to future.
	err = store.SetLastVulnerabilityUpdate(ctx, "valid", now.Add(time.Hour))
	require.NoError(t, err)

	// Expect the new timestamp inserted before to be used.
	timestamp, err = store.GetLastVulnerabilityUpdate(ctx)
	require.NoError(t, err)
	assert.Equal(t, now.UTC().Truncate(time.Second), timestamp)
}
