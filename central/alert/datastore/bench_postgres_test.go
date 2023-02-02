//go:build sql_integration
// +build sql_integration

package datastore

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stackrox/rox/central/alert/datastore/internal/search"
	postgresStore "github.com/stackrox/rox/central/alert/datastore/internal/store/postgres"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/require"
)

func BenchmarkDBsWithPostgres(b *testing.B) {
	b.Setenv(env.PostgresDatastoreEnabled.EnvVar(), "true")
	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		b.Skipf("%q not set. Skip postgres test", env.PostgresDatastoreEnabled.EnvVar())
		b.SkipNow()
	}

	ctx := sac.WithAllAccess(context.Background())
	source := pgtest.GetConnectionString(b)
	config, err := pgxpool.ParseConfig(source)
	require.NoError(b, err)
	db, err := pgxpool.NewWithConfig(ctx, config)
	require.NoError(b, err)
	gormDB := pgtest.OpenGormDB(b, source)
	defer pgtest.CloseGormDB(b, gormDB)

	postgresStore.Destroy(ctx, db)
	store := postgresStore.CreateTableAndNewStore(ctx, db, gormDB)
	indexer := postgresStore.NewIndexer(db)
	datastore, err := New(store, indexer, search.New(store, indexer))
	require.NoError(b, err)

	var ids []string
	for i := 0; i < 15000; i++ {
		id := fmt.Sprintf("%d", i)
		ids = append(ids, id)
		a := fixtures.GetAlertWithID(id)
		require.NoError(b, store.Upsert(ctx, a))
	}

	log.Info("Successfully loaded the DB")

	b.Run("markStale", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, id := range ids {
				require.NoError(b, datastore.MarkAlertStale(ctx, id))
			}
		}
	})

	b.Run("markStaleBatch", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := datastore.MarkAlertStaleBatch(ctx, ids...)
			require.NoError(b, err)
		}
	})
}
