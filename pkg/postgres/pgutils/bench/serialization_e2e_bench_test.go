//go:build sql_integration

package bench

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"google.golang.org/protobuf/encoding/protojson"
)

// BenchmarkE2EBytea benchmarks INSERT+SELECT round-trip using a bytea column.
func BenchmarkE2EBytea(b *testing.B) {
	tp := pgtest.ForT(b)
	ctx := sac.WithAllAccess(context.Background())

	_, err := tp.Exec(ctx, `CREATE TABLE IF NOT EXISTS bench_bytea (id TEXT PRIMARY KEY, serialized bytea)`)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { _, _ = tp.Exec(ctx, `DROP TABLE IF EXISTS bench_bytea`) })

	deployment := fixtures.GetDeployment()
	data, err := deployment.MarshalVT()
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := fmt.Sprintf("id-%d", i)
		_, err := tp.Exec(ctx, `INSERT INTO bench_bytea (id, serialized) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET serialized = $2`, id, data)
		if err != nil {
			b.Fatal(err)
		}

		var readBack []byte
		err = tp.QueryRow(ctx, `SELECT serialized FROM bench_bytea WHERE id = $1`, id).Scan(&readBack)
		if err != nil {
			b.Fatal(err)
		}

		var msg storage.Deployment
		if err := msg.UnmarshalVTUnsafe(readBack); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkE2EJsonb benchmarks INSERT+SELECT round-trip using a jsonb column.
func BenchmarkE2EJsonb(b *testing.B) {
	tp := pgtest.ForT(b)
	ctx := sac.WithAllAccess(context.Background())

	_, err := tp.Exec(ctx, `CREATE TABLE IF NOT EXISTS bench_jsonb (id TEXT PRIMARY KEY, serialized jsonb)`)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { _, _ = tp.Exec(ctx, `DROP TABLE IF EXISTS bench_jsonb`) })

	deployment := fixtures.GetDeployment()
	data, err := protojson.Marshal(deployment)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := fmt.Sprintf("id-%d", i)
		_, err := tp.Exec(ctx, `INSERT INTO bench_jsonb (id, serialized) VALUES ($1, $2::jsonb) ON CONFLICT (id) DO UPDATE SET serialized = $2::jsonb`, id, data)
		if err != nil {
			b.Fatal(err)
		}

		var readBack []byte
		err = tp.QueryRow(ctx, `SELECT serialized FROM bench_jsonb WHERE id = $1`, id).Scan(&readBack)
		if err != nil {
			b.Fatal(err)
		}

		var msg storage.Deployment
		if err := protojson.Unmarshal(readBack, &msg); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkE2EBulkWrite benchmarks CopyFrom performance for bytea vs jsonb.
func BenchmarkE2EBulkWrite(b *testing.B) {
	tp := pgtest.ForT(b)
	ctx := sac.WithAllAccess(context.Background())

	deployment := fixtures.GetDeployment()
	const rowCount = 1000

	b.Run("Bytea", func(b *testing.B) {
		_, err := tp.Exec(ctx, `CREATE TABLE IF NOT EXISTS bench_bulk_bytea (id TEXT PRIMARY KEY, serialized bytea)`)
		if err != nil {
			b.Fatal(err)
		}
		b.Cleanup(func() { _, _ = tp.Exec(ctx, `DROP TABLE IF EXISTS bench_bulk_bytea`) })

		data, err := deployment.MarshalVT()
		if err != nil {
			b.Fatal(err)
		}

		rows := make([][]interface{}, rowCount)
		for i := 0; i < rowCount; i++ {
			rows[i] = []interface{}{fmt.Sprintf("id-%d", i), data}
		}

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = tp.Exec(ctx, `TRUNCATE bench_bulk_bytea`)
			tx, err := tp.Begin(ctx)
			if err != nil {
				b.Fatal(err)
			}
			_, err = tx.CopyFrom(ctx,
				pgx.Identifier{"bench_bulk_bytea"},
				[]string{"id", "serialized"},
				pgx.CopyFromRows(rows),
			)
			if err != nil {
				_ = tx.Rollback(ctx)
				b.Fatal(err)
			}
			if err := tx.Commit(ctx); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Jsonb", func(b *testing.B) {
		_, err := tp.Exec(ctx, `CREATE TABLE IF NOT EXISTS bench_bulk_jsonb (id TEXT PRIMARY KEY, serialized jsonb)`)
		if err != nil {
			b.Fatal(err)
		}
		b.Cleanup(func() { _, _ = tp.Exec(ctx, `DROP TABLE IF EXISTS bench_bulk_jsonb`) })

		data, err := protojson.Marshal(deployment)
		if err != nil {
			b.Fatal(err)
		}

		rows := make([][]interface{}, rowCount)
		for i := 0; i < rowCount; i++ {
			rows[i] = []interface{}{fmt.Sprintf("id-%d", i), data}
		}

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = tp.Exec(ctx, `TRUNCATE bench_bulk_jsonb`)
			tx, err := tp.Begin(ctx)
			if err != nil {
				b.Fatal(err)
			}
			_, err = tx.CopyFrom(ctx,
				pgx.Identifier{"bench_bulk_jsonb"},
				[]string{"id", "serialized"},
				pgx.CopyFromRows(rows),
			)
			if err != nil {
				_ = tx.Rollback(ctx)
				b.Fatal(err)
			}
			if err := tx.Commit(ctx); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkE2EBulkRead benchmarks bulk SELECT performance for bytea vs jsonb.
func BenchmarkE2EBulkRead(b *testing.B) {
	tp := pgtest.ForT(b)
	ctx := sac.WithAllAccess(context.Background())

	deployment := fixtures.GetDeployment()
	const rowCount = 1000

	b.Run("Bytea", func(b *testing.B) {
		_, err := tp.Exec(ctx, `CREATE TABLE IF NOT EXISTS bench_read_bytea (id TEXT PRIMARY KEY, serialized bytea)`)
		if err != nil {
			b.Fatal(err)
		}
		b.Cleanup(func() { _, _ = tp.Exec(ctx, `DROP TABLE IF EXISTS bench_read_bytea`) })

		data, err := deployment.MarshalVT()
		if err != nil {
			b.Fatal(err)
		}

		// Seed the table using a transaction with CopyFrom.
		tx, err := tp.Begin(ctx)
		if err != nil {
			b.Fatal(err)
		}
		rows := make([][]interface{}, rowCount)
		for i := 0; i < rowCount; i++ {
			rows[i] = []interface{}{fmt.Sprintf("id-%d", i), data}
		}
		_, err = tx.CopyFrom(ctx, pgx.Identifier{"bench_read_bytea"}, []string{"id", "serialized"}, pgx.CopyFromRows(rows))
		if err != nil {
			_ = tx.Rollback(ctx)
			b.Fatal(err)
		}
		if err := tx.Commit(ctx); err != nil {
			b.Fatal(err)
		}

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dbRows, err := tp.Query(ctx, `SELECT serialized FROM bench_read_bytea`)
			if err != nil {
				b.Fatal(err)
			}
			for dbRows.Next() {
				var serialized []byte
				if err := dbRows.Scan(&serialized); err != nil {
					b.Fatal(err)
				}
				var msg storage.Deployment
				if err := msg.UnmarshalVTUnsafe(serialized); err != nil {
					b.Fatal(err)
				}
			}
			dbRows.Close()
			if err := dbRows.Err(); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Jsonb", func(b *testing.B) {
		_, err := tp.Exec(ctx, `CREATE TABLE IF NOT EXISTS bench_read_jsonb (id TEXT PRIMARY KEY, serialized jsonb)`)
		if err != nil {
			b.Fatal(err)
		}
		b.Cleanup(func() { _, _ = tp.Exec(ctx, `DROP TABLE IF EXISTS bench_read_jsonb`) })

		data, err := protojson.Marshal(deployment)
		if err != nil {
			b.Fatal(err)
		}

		// Seed the table using a transaction with CopyFrom.
		tx, err := tp.Begin(ctx)
		if err != nil {
			b.Fatal(err)
		}
		rows := make([][]interface{}, rowCount)
		for i := 0; i < rowCount; i++ {
			rows[i] = []interface{}{fmt.Sprintf("id-%d", i), data}
		}
		_, err = tx.CopyFrom(ctx, pgx.Identifier{"bench_read_jsonb"}, []string{"id", "serialized"}, pgx.CopyFromRows(rows))
		if err != nil {
			_ = tx.Rollback(ctx)
			b.Fatal(err)
		}
		if err := tx.Commit(ctx); err != nil {
			b.Fatal(err)
		}

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dbRows, err := tp.Query(ctx, `SELECT serialized FROM bench_read_jsonb`)
			if err != nil {
				b.Fatal(err)
			}
			for dbRows.Next() {
				var serialized []byte
				if err := dbRows.Scan(&serialized); err != nil {
					b.Fatal(err)
				}
				var msg storage.Deployment
				if err := protojson.Unmarshal(serialized, &msg); err != nil {
					b.Fatal(err)
				}
			}
			dbRows.Close()
			if err := dbRows.Err(); err != nil {
				b.Fatal(err)
			}
		}
	})
}
