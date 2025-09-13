//go:build scanner_db_integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/quay/claircore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ExternalIndexReport_GCIndexReports(t *testing.T) {
	ctx := context.Background()
	pool := testDB(t, ctx, "external_index_report_test")
	store, err := InitPostgresExternalIndexStore(ctx, pool)
	require.NoError(t, err)
	now := time.Now()

	ir := &claircore.IndexReport{State: "sample state", Success: true}

	// Add an index report which should not be deleted.
	err = store.StoreIndexReport(ctx, "sha512:abc", ir, now.Add(1*time.Hour))
	assert.NoError(t, err)
	// Run GC but do not delete.
	ids, err := store.GCIndexReports(ctx, now)
	require.NoError(t, err)
	assert.Len(t, ids, 0)

	// Add an index report to be deleted.
	// First, add it an hour ahead.
	err = store.StoreIndexReport(ctx, "sha512:def", ir, now.Add(1*time.Hour))
	assert.NoError(t, err)
	// Next, add it an hour behind.
	// This ensures the row is overwritten.
	err = store.StoreIndexReport(ctx, "sha512:def", ir, now.Add(-1*time.Hour))
	assert.NoError(t, err)
	// Delete the index report.
	ids, err = store.GCIndexReports(ctx, now)
	require.NoError(t, err)
	assert.Len(t, ids, 1)
	assert.Equal(t, "sha512:def", ids[0])

	// Add an index report which manages to have the same time as now.
	err = store.StoreIndexReport(ctx, "sha512:ghi", ir, now)
	assert.NoError(t, err)
	// This index report gets lucky, and it does not get deleted. Yet...
	// Note the index report from the previous test is also not deleted (again).
	ids, err = store.GCIndexReports(ctx, now)
	require.NoError(t, err)
	assert.Len(t, ids, 0)
}
