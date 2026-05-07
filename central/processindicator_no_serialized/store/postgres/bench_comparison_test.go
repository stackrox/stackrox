//go:build sql_integration

package postgres

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
)

// Tier 2: Postgres-side analysis

func TestPgInternals_StorageSize(t *testing.T) {
	ctx := sac.WithAllAccess(context.Background())
	db := pgtest.ForT(t)
	store := New(db.DB)

	for _, rowCount := range []int{5000, 100000} {
		t.Run(fmt.Sprintf("rows_%d", rowCount), func(t *testing.T) {
			_, err := db.DB.Exec(ctx, "TRUNCATE process_indicator_no_serializeds CASCADE")
			require.NoError(t, err)

			batch := make([]*storeType, 0, 500)
			for range rowCount {
				batch = append(batch, makeNoSerializedIndicator(uuid.NewV4().String()))
				if len(batch) == 500 {
					require.NoError(t, store.UpsertMany(ctx, batch))
					batch = batch[:0]
				}
			}
			if len(batch) > 0 {
				require.NoError(t, store.UpsertMany(ctx, batch))
			}

			// Analyze table for accurate stats
			_, err = db.DB.Exec(ctx, "ANALYZE process_indicator_no_serializeds")
			require.NoError(t, err)

			// Measure sizes
			var totalSize, tableSize, indexSize int64
			err = db.DB.QueryRow(ctx,
				`SELECT pg_total_relation_size('process_indicator_no_serializeds'),
				        pg_table_size('process_indicator_no_serializeds'),
				        pg_indexes_size('process_indicator_no_serializeds')`,
			).Scan(&totalSize, &tableSize, &indexSize)
			require.NoError(t, err)

			// Toast size
			var toastSize int64
			err = db.DB.QueryRow(ctx,
				`SELECT COALESCE(pg_total_relation_size(reltoastrelid), 0)
				 FROM pg_class WHERE relname = 'process_indicator_no_serializeds'`,
			).Scan(&toastSize)
			require.NoError(t, err)

			// Average row size
			var avgRowSize float64
			err = db.DB.QueryRow(ctx,
				`SELECT avg(pg_column_size(t.*)) FROM process_indicator_no_serializeds t LIMIT 1000`,
			).Scan(&avgRowSize)
			require.NoError(t, err)

			t.Logf("=== STORAGE SIZE (rows=%d) ===", rowCount)
			t.Logf("Total:    %d KB", totalSize/1024)
			t.Logf("Table:    %d KB", tableSize/1024)
			t.Logf("Index:    %d KB", indexSize/1024)
			t.Logf("Toast:    %d KB", toastSize/1024)
			t.Logf("Avg row:  %.0f B", avgRowSize)
		})
	}
}

func TestPgInternals_ExplainAnalyze(t *testing.T) {
	ctx := sac.WithAllAccess(context.Background())
	db := pgtest.ForT(t)
	store := New(db.DB)

	// Insert 5000 rows
	for range 10 {
		batch := make([]*storeType, 500)
		for j := range batch {
			batch[j] = makeNoSerializedIndicator(uuid.NewV4().String())
		}
		require.NoError(t, store.UpsertMany(ctx, batch))
	}
	_, err := db.DB.Exec(ctx, "ANALYZE process_indicator_no_serializeds")
	require.NoError(t, err)

	queries := map[string]string{
		"PKLookup": "EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT) SELECT * FROM process_indicator_no_serializeds WHERE id = '" + uuid.NewV4().String() + "'",
		"FullScan": "EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT) SELECT * FROM process_indicator_no_serializeds",
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			rows, err := db.DB.Query(ctx, query)
			require.NoError(t, err)
			defer rows.Close()
			for rows.Next() {
				var line string
				require.NoError(t, rows.Scan(&line))
				t.Log(line)
			}
		})
	}

	// IN-list with 500 existing IDs
	t.Run("INList500", func(t *testing.T) {
		// Get 500 actual IDs
		rows, err := db.DB.Query(ctx, "SELECT id FROM process_indicator_no_serializeds LIMIT 500")
		require.NoError(t, err)
		var ids []string
		for rows.Next() {
			var id string
			require.NoError(t, rows.Scan(&id))
			ids = append(ids, "'"+id+"'")
		}
		rows.Close()

		query := fmt.Sprintf("EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT) SELECT * FROM process_indicator_no_serializeds WHERE id IN (%s)", strings.Join(ids, ","))
		explainRows, err := db.DB.Query(ctx, query)
		require.NoError(t, err)
		defer explainRows.Close()
		for explainRows.Next() {
			var line string
			require.NoError(t, explainRows.Scan(&line))
			t.Log(line)
		}
	})
}

func TestPgInternals_WALVolume(t *testing.T) {
	ctx := sac.WithAllAccess(context.Background())
	db := pgtest.ForT(t)
	store := New(db.DB)

	// Checkpoint for clean measurement
	_, err := db.DB.Exec(ctx, "CHECKPOINT")
	require.NoError(t, err)

	var walBefore int64
	err = db.DB.QueryRow(ctx, "SELECT pg_wal_lsn_diff(pg_current_wal_lsn(), '0/0')").Scan(&walBefore)
	require.NoError(t, err)

	// Insert 10K rows
	for range 20 {
		batch := make([]*storeType, 500)
		for j := range batch {
			batch[j] = makeNoSerializedIndicator(uuid.NewV4().String())
		}
		require.NoError(t, store.UpsertMany(ctx, batch))
	}

	var walAfter int64
	err = db.DB.QueryRow(ctx, "SELECT pg_wal_lsn_diff(pg_current_wal_lsn(), '0/0')").Scan(&walAfter)
	require.NoError(t, err)

	walBytes := walAfter - walBefore
	t.Logf("=== WAL VOLUME ===")
	t.Logf("WAL for 10K upserts: %d KB (%.0f bytes/row)", walBytes/1024, float64(walBytes)/10000)
}

// Tier 3: Scale benchmarks

func BenchmarkScale_GetMany_100K(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())
	db := pgtest.ForT(b)
	store := New(db.DB)

	// Pre-populate 100K rows
	ids := make([]string, 0, 100000)
	for range 200 {
		batch := make([]*storeType, 500)
		for j := range batch {
			batch[j] = makeNoSerializedIndicator(uuid.NewV4().String())
			ids = append(ids, batch[j].GetId())
		}
		if err := store.UpsertMany(ctx, batch); err != nil {
			b.Fatal(err)
		}
	}

	// Benchmark GetMany of 500 from 100K table
	readIDs := ids[:500]
	b.ResetTimer()
	for b.Loop() {
		_, _, err := store.GetMany(ctx, readIDs)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkScale_UpsertInto100K(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())
	db := pgtest.ForT(b)
	store := New(db.DB)

	// Pre-populate 100K rows
	for range 200 {
		batch := make([]*storeType, 500)
		for j := range batch {
			batch[j] = makeNoSerializedIndicator(uuid.NewV4().String())
		}
		if err := store.UpsertMany(ctx, batch); err != nil {
			b.Fatal(err)
		}
	}

	// Benchmark inserting 500 new rows into 100K table
	b.ResetTimer()
	for b.Loop() {
		batch := make([]*storeType, 500)
		for j := range batch {
			batch[j] = makeNoSerializedIndicator(uuid.NewV4().String())
		}
		if err := store.UpsertMany(ctx, batch); err != nil {
			b.Fatal(err)
		}
	}
}
