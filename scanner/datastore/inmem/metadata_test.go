package inmem

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetadataStore(t *testing.T) {
	ctx := context.Background()
	m := NewMetadataStore()

	// Empty store reports zero time (forces initial bundle import on boot).
	ts, err := m.GetLastVulnerabilityUpdate(ctx)
	require.NoError(t, err)
	assert.True(t, ts.IsZero())

	t1 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 1, 1, 14, 0, 0, 0, time.UTC)

	got, err := m.GetOrSetLastVulnerabilityUpdate(ctx, "rhel-vex", t2)
	require.NoError(t, err)
	assert.Equal(t, t2, got)

	// Existing entry is returned unchanged by GetOrSet.
	got, err = m.GetOrSetLastVulnerabilityUpdate(ctx, "rhel-vex", t3)
	require.NoError(t, err)
	assert.Equal(t, t2, got)

	require.NoError(t, m.SetLastVulnerabilityUpdate(ctx, "alpine", t1))
	require.NoError(t, m.SetLastVulnerabilityUpdate(ctx, "stale-bundle", t1))

	// GetLastVulnerabilityUpdate returns the oldest timestamp.
	oldest, err := m.GetLastVulnerabilityUpdate(ctx)
	require.NoError(t, err)
	assert.Equal(t, t1, oldest)

	// Advance alpine and GC stale-bundle.
	require.NoError(t, m.SetLastVulnerabilityUpdate(ctx, "alpine", t3))
	require.NoError(t, m.GCVulnerabilityUpdates(ctx, []string{"rhel-vex", "alpine"}, t2))

	oldest, err = m.GetLastVulnerabilityUpdate(ctx)
	require.NoError(t, err)
	assert.Equal(t, t2, oldest)

	// A pre-registered bundle with zero timestamp must keep oldest at zero
	// until every bundle finishes importing (mirrors Postgres ORDER BY LIMIT 1).
	_, err = m.GetOrSetLastVulnerabilityUpdate(ctx, "pending-bundle", time.Time{})
	require.NoError(t, err)
	oldest, err = m.GetLastVulnerabilityUpdate(ctx)
	require.NoError(t, err)
	assert.True(t, oldest.IsZero(), "zero timestamp from pre-registered bundle must not be skipped")
}
