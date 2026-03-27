//go:build sql_integration

package postgres

import (
	"context"
	"fmt"
	"testing"

	serializedStore "github.com/stackrox/rox/central/processindicator/store/postgres"
	noSerializedStore "github.com/stackrox/rox/central/processindicator_no_serialized/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
)

// TestPostgresInternals seeds each store with the same logical data, then
// queries Postgres catalog views to compare storage footprint and
// server-side query performance across serialized (bytea), jsonb, and
// no-serialized store strategies.
func TestPostgresInternals(t *testing.T) {
	ctx := sac.WithAllAccess(context.Background())
	db := pgtest.ForT(t)

	const seedCount = 5000

	// --- seed stores ---
	sSt := serializedStore.New(db.DB)
	jSt := New(db.DB)
	nSt := noSerializedStore.New(db.DB)

	sObjs := make([]*storage.ProcessIndicator, seedCount)
	jObjs := make([]*storage.ProcessIndicatorJsonb, seedCount)
	nObjs := make([]*storage.ProcessIndicatorNoSerialized, seedCount)
	for i := 0; i < seedCount; i++ {
		id := uuid.NewV4().String()
		sObjs[i] = makeSerializedIndicator(id)
		jObjs[i] = makeJsonbIndicator(uuid.NewV4().String())
		nObjs[i] = makeNoSerializedIndicator(uuid.NewV4().String())
	}
	require.NoError(t, sSt.UpsertMany(ctx, sObjs))
	require.NoError(t, jSt.UpsertMany(ctx, jObjs))
	require.NoError(t, nSt.UpsertMany(ctx, nObjs))

	// Force Postgres to update statistics
	_, err := db.DB.Exec(ctx, "ANALYZE process_indicators")
	require.NoError(t, err)
	_, err = db.DB.Exec(ctx, "ANALYZE process_indicator_jsonbs")
	require.NoError(t, err)
	_, err = db.DB.Exec(ctx, "ANALYZE process_indicator_no_serializeds")
	require.NoError(t, err)

	t.Log("\n========== STORAGE SIZE ==========")

	type sizeResult struct {
		table     string
		totalSize string
		tableSize string
		indexSize string
		rowCount  int
	}

	for _, tbl := range []string{
		"process_indicators",
		"process_indicator_jsonbs",
		"process_indicator_no_serializeds",
	} {
		var r sizeResult
		r.table = tbl
		err := db.DB.QueryRow(ctx, fmt.Sprintf(`
			SELECT
				pg_size_pretty(pg_total_relation_size('%s')) AS total,
				pg_size_pretty(pg_table_size('%s')) AS tbl,
				pg_size_pretty(pg_indexes_size('%s')) AS idx,
				(SELECT count(*) FROM %s)::int AS cnt
		`, tbl, tbl, tbl, tbl)).Scan(&r.totalSize, &r.tableSize, &r.indexSize, &r.rowCount)
		require.NoError(t, err)
		t.Logf("  %-40s  rows=%-5d  total=%-10s  table=%-10s  indexes=%s",
			r.table, r.rowCount, r.totalSize, r.tableSize, r.indexSize)
	}

	// Also show child table for no-serialized
	{
		var total, tbl, idx string
		var cnt int
		err := db.DB.QueryRow(ctx, `
			SELECT
				pg_size_pretty(pg_total_relation_size('process_indicator_no_serializeds_lineage_infos')),
				pg_size_pretty(pg_table_size('process_indicator_no_serializeds_lineage_infos')),
				pg_size_pretty(pg_indexes_size('process_indicator_no_serializeds_lineage_infos')),
				(SELECT count(*) FROM process_indicator_no_serializeds_lineage_infos)::int
		`).Scan(&total, &tbl, &idx, &cnt)
		require.NoError(t, err)
		t.Logf("  %-40s  rows=%-5d  total=%-10s  table=%-10s  indexes=%s",
			"  └─ _lineage_infos (child)", cnt, total, tbl, idx)
	}

	t.Log("\n========== AVERAGE ROW / COLUMN SIZE (bytes) ==========")

	// Serialized (bytea): show avg size of the serialized column vs total row
	{
		var avgRow, avgSerialized float64
		err := db.DB.QueryRow(ctx, `
			SELECT
				avg(pg_column_size(t.*))::numeric(10,1),
				avg(pg_column_size(t.serialized))::numeric(10,1)
			FROM process_indicators t
		`).Scan(&avgRow, &avgSerialized)
		require.NoError(t, err)
		t.Logf("  Serialized (bytea):      avg_row=%.0f B   avg_serialized_col=%.0f B  (%.0f%% of row)",
			avgRow, avgSerialized, avgSerialized/avgRow*100)
	}

	// Jsonb: show avg size of the serialized jsonb column vs total row
	{
		var avgRow, avgSerialized float64
		err := db.DB.QueryRow(ctx, `
			SELECT
				avg(pg_column_size(t.*))::numeric(10,1),
				avg(pg_column_size(t.serialized))::numeric(10,1)
			FROM process_indicator_jsonbs t
		`).Scan(&avgRow, &avgSerialized)
		require.NoError(t, err)
		t.Logf("  Jsonb:                   avg_row=%.0f B   avg_serialized_col=%.0f B  (%.0f%% of row)",
			avgRow, avgSerialized, avgSerialized/avgRow*100)
	}

	// NoSerialized: no blob column, show total row size
	{
		var avgRow float64
		err := db.DB.QueryRow(ctx, `
			SELECT avg(pg_column_size(t.*))::numeric(10,1)
			FROM process_indicator_no_serializeds t
		`).Scan(&avgRow)
		require.NoError(t, err)
		t.Logf("  NoSerialized:            avg_row=%.0f B   (no blob column)", avgRow)
	}

	t.Log("\n========== EXPLAIN ANALYZE: GET single row by PK ==========")

	// Pick one ID from each table
	var sID, jID, nID string
	require.NoError(t, db.DB.QueryRow(ctx, "SELECT id FROM process_indicators LIMIT 1").Scan(&sID))
	require.NoError(t, db.DB.QueryRow(ctx, "SELECT id FROM process_indicator_jsonbs LIMIT 1").Scan(&jID))
	require.NoError(t, db.DB.QueryRow(ctx, "SELECT id FROM process_indicator_no_serializeds LIMIT 1").Scan(&nID))

	for _, tc := range []struct {
		label string
		query string
		id    string
	}{
		{
			"Serialized (bytea) — select serialized",
			"SELECT serialized FROM process_indicators WHERE id = $1",
			sID,
		},
		{
			"Jsonb — select serialized",
			"SELECT serialized FROM process_indicator_jsonbs WHERE id = $1",
			jID,
		},
		{
			"NoSerialized — select all columns",
			"SELECT * FROM process_indicator_no_serializeds WHERE id = $1",
			nID,
		},
	} {
		rows, err := db.DB.Query(ctx, "EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT) "+tc.query, tc.id)
		require.NoError(t, err)
		t.Logf("  --- %s ---", tc.label)
		for rows.Next() {
			var line string
			require.NoError(t, rows.Scan(&line))
			t.Logf("    %s", line)
		}
		rows.Close()
	}

	t.Log("\n========== EXPLAIN ANALYZE: GET 500 rows by PK list ==========")

	// Collect 500 IDs from each table
	sIDs := collectIDs(t, ctx, db, "process_indicators", 500)
	jIDs := collectIDs(t, ctx, db, "process_indicator_jsonbs", 500)
	nIDs := collectIDs(t, ctx, db, "process_indicator_no_serializeds", 500)

	for _, tc := range []struct {
		label string
		query string
		ids   []string
	}{
		{
			"Serialized (bytea) — select serialized WHERE id = ANY",
			"SELECT serialized FROM process_indicators WHERE id = ANY($1::uuid[])",
			sIDs,
		},
		{
			"Jsonb — select serialized WHERE id = ANY",
			"SELECT serialized FROM process_indicator_jsonbs WHERE id = ANY($1::uuid[])",
			jIDs,
		},
		{
			"NoSerialized — select * WHERE id = ANY",
			"SELECT * FROM process_indicator_no_serializeds WHERE id = ANY($1::uuid[])",
			nIDs,
		},
	} {
		rows, err := db.DB.Query(ctx, "EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT) "+tc.query, tc.ids)
		require.NoError(t, err)
		t.Logf("  --- %s ---", tc.label)
		for rows.Next() {
			var line string
			require.NoError(t, rows.Scan(&line))
			t.Logf("    %s", line)
		}
		rows.Close()
	}

	t.Log("\n========== EXPLAIN ANALYZE: seq scan (Walk) ==========")

	for _, tc := range []struct {
		label string
		query string
	}{
		{"Serialized (bytea)", "SELECT serialized FROM process_indicators"},
		{"Jsonb", "SELECT serialized FROM process_indicator_jsonbs"},
		{"NoSerialized", "SELECT * FROM process_indicator_no_serializeds"},
	} {
		rows, err := db.DB.Query(ctx, "EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT) "+tc.query)
		require.NoError(t, err)
		t.Logf("  --- %s ---", tc.label)
		for rows.Next() {
			var line string
			require.NoError(t, rows.Scan(&line))
			t.Logf("    %s", line)
		}
		rows.Close()
	}
}

func collectIDs(t *testing.T, ctx context.Context, db *pgtest.TestPostgres, table string, n int) []string {
	rows, err := db.DB.Query(ctx, fmt.Sprintf("SELECT id FROM %s LIMIT %d", table, n))
	require.NoError(t, err)
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		require.NoError(t, rows.Scan(&id))
		ids = append(ids, id)
	}
	return ids
}
