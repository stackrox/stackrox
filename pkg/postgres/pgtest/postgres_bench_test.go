//go:build sql_integration

package pgtest

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/lib/pq"
	"github.com/stackrox/rox/pkg/postgres/pgtest/conn"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stretchr/testify/require"
)

func BenchmarkForT(b *testing.B) {
	for b.Loop() {
		tp := ForT(b)
		_ = tp
	}
}

func BenchmarkCreateFromTemplate(b *testing.B) {
	const tmpl = "bench_template"
	CreateDatabase(b, tmpl)
	src := conn.GetConnectionStringWithDatabaseName(b, tmpl)
	gormDB := OpenGormDB(b, src)
	pkgSchema.ApplyAllSchemasIncludingTests(context.Background(), gormDB, b)
	CloseGormDB(b, gormDB)
	b.Cleanup(func() { DropDatabase(b, tmpl) })

	var i int
	for b.Loop() {
		i++
		dbName := fmt.Sprintf("bench_tmpl_%d", i)
		adminSrc := conn.GetConnectionStringWithDatabaseName(b, defaultDatabaseName)
		db, err := sql.Open(driverName, adminSrc)
		require.NoError(b, err)
		_, err = db.Exec("CREATE DATABASE " + pq.QuoteIdentifier(dbName) + " TEMPLATE " + pq.QuoteIdentifier(tmpl))
		require.NoError(b, err)
		require.NoError(b, db.Close())
		b.Cleanup(func() { DropDatabase(b, dbName) })
	}
}
