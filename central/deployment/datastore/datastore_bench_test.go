package datastore

import (
	"context"
	"fmt"
	"testing"

	pgStore "github.com/stackrox/rox/central/deployment/datastore/internal/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func BenchmarkSearchAllDeployments(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())
	testDB := pgtest.ForT(b)

	deploymentsDatastore, err := GetTestPostgresDataStore(b, testDB.DB)
	require.NoError(b, err)

	deploymentPrototype := fixtures.GetDeployment().CloneVT()
	const numDeployments = 1000
	for i := 0; i < numDeployments; i++ {
		if i > 0 && i%100 == 0 {
			fmt.Println("Added", i, "deployments")
		}
		deploymentPrototype.Id = uuid.NewV4().String()
		require.NoError(b, deploymentsDatastore.UpsertDeployment(ctx, deploymentPrototype))
	}

	b.Run("SearchRetrievalList", func(b *testing.B) {
		for b.Loop() {
			deployments, err := deploymentsDatastore.SearchListDeployments(ctx, search.EmptyQuery())
			assert.NoError(b, err)
			assert.Len(b, deployments, numDeployments)
		}
	})

	b.Run("SearchRetrievalFull", func(b *testing.B) {
		for b.Loop() {
			deployments, err := deploymentsDatastore.SearchRawDeployments(ctx, search.EmptyQuery())
			assert.NoError(b, err)
			assert.Len(b, deployments, numDeployments)
		}
	})
}

// seedDeployments bulk-inserts deployments directly via the postgres store,
// bypassing the datastore layer (rankings, platform matching) for speed.
// It creates a mix of active and soft-deleted deployments according to
// deletedPercent (0–100).
func seedDeployments(b *testing.B, ctx context.Context, store pgStore.Store, total int, deletedPercent int) {
	b.Helper()

	const batchSize = 500
	prototype := fixtures.GetDeployment().CloneVT()

	batch := make([]*storage.Deployment, 0, batchSize)
	for i := 0; i < total; i++ {
		d := prototype.CloneVT()
		d.Id = uuid.NewV4().String()

		if i%100 < deletedPercent {
			d.State = storage.DeploymentState_STATE_DELETED
			d.Deleted = protocompat.TimestampNow()
		} else {
			d.State = storage.DeploymentState_STATE_ACTIVE
			d.Deleted = nil
		}

		batch = append(batch, d)
		if len(batch) == batchSize {
			require.NoError(b, store.UpsertMany(ctx, batch))
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		require.NoError(b, store.UpsertMany(ctx, batch))
	}
}

// BenchmarkSoftDeleteQueries measures the performance of queries that filter
// deployments by their soft-delete state under four index configurations:
//
//   - DeletedOnly:  index on (deleted) — the current migration index.
//   - StateOnly:    index on (state).
//   - Composite:    index on (state, deleted).
//   - NoIndex:      no additional index beyond the primary key.
//
// Query patterns benchmarked:
//   - CountActive:      COUNT with state = STATE_ACTIVE (hot path for every list/search).
//   - SearchActiveList: full list retrieval filtered to active deployments.
//   - CountDeleted:     COUNT with state = STATE_DELETED.
//   - PruneQuery:       deleted < retention_window (the garbage-collector query).
func BenchmarkSoftDeleteQueries(b *testing.B) {
	b.Setenv(features.DeploymentSoftDeletion.EnvVar(), "true")

	scales := []int{25_000, 100_000, 250_000}

	for _, n := range scales {
		b.Run(fmt.Sprintf("N=%d", n), func(b *testing.B) {
			ctx := sac.WithAllAccess(context.Background())
			testDB := pgtest.ForT(b)

			store := pgStore.New(testDB.DB)

			// Seed with 90% active, 10% soft-deleted.
			b.Logf("Seeding %d deployments...", n)
			seedDeployments(b, ctx, store, n, 10)
			b.Logf("Seeding complete.")

			// activeQuery filters for active deployments, the most common query path.
			activeQuery := search.NewQueryBuilder().
				AddExactMatches(search.DeploymentState, storage.DeploymentState_STATE_ACTIVE.String()).
				ProtoQuery()

			// pruneQuery filters by the deleted timestamp only. If deleted is set,
			// the state is assumed to be STATE_DELETED, so a conjunction on state
			// is redundant.
			pruneQuery := search.NewQueryBuilder().
				AddDays(search.Deleted, 0). // All soft-deleted deployments (retention = 0 days).
				ProtoQuery()

			// countDeletedQuery counts all soft-deleted deployments.
			countDeletedQuery := search.NewQueryBuilder().
				AddExactMatches(search.DeploymentState, storage.DeploymentState_STATE_DELETED.String()).
				ProtoQuery()

			// warmupQueries exercises every query path to populate PostgreSQL
			// shared buffers and OS page cache before measurement begins.
			warmupQueries := func() {
				const iterations = 20
				for range iterations {
					store.Count(ctx, activeQuery)       //nolint:errcheck
					store.Search(ctx, activeQuery)      //nolint:errcheck
					store.Count(ctx, countDeletedQuery) //nolint:errcheck
					store.Search(ctx, pruneQuery)       //nolint:errcheck
				}
			}

			// All benchmarks use the postgres store directly to measure SQL
			// query performance without Go-side overhead (ranking, etc.).
			// Using the datastore's SearchListDeployments would trigger the
			// ranker's dev-mode mutex timeout at large scales.
			runBenchmarks := func(b *testing.B, label string) {
				b.Run(label+"/CountActive", func(b *testing.B) {
					for b.Loop() {
						count, err := store.Count(ctx, activeQuery)
						assert.NoError(b, err)
						assert.Greater(b, count, 0)
					}
				})

				b.Run(label+"/SearchActive", func(b *testing.B) {
					for b.Loop() {
						results, err := store.Search(ctx, activeQuery)
						assert.NoError(b, err)
						assert.Greater(b, len(results), 0)
					}
				})

				b.Run(label+"/CountDeleted", func(b *testing.B) {
					for b.Loop() {
						count, err := store.Count(ctx, countDeletedQuery)
						assert.NoError(b, err)
						assert.Greater(b, count, 0)
					}
				})

				b.Run(label+"/PruneQuery", func(b *testing.B) {
					for b.Loop() {
						// Use Search instead of PurgeDeployments to measure query
						// performance without actually deleting data.
						results, err := store.Search(ctx, pruneQuery)
						assert.NoError(b, err)
						assert.Greater(b, len(results), 0)
					}
				})
			}

			// dropAllSoftDeleteIndexes removes all custom indexes so each
			// configuration starts from a clean slate.
			dropAllSoftDeleteIndexes := func() {
				for _, idx := range []string{
					"deployments_deleted",
					"deployments_state",
					"deployments_state_deleted",
				} {
					_, err := testDB.DB.Exec(ctx, fmt.Sprintf("DROP INDEX IF EXISTS %s", idx))
					require.NoError(b, err)
				}
			}

			// --- Index configuration: deleted only (current migration) ---
			// The schema migration already created deployments_deleted.
			// Drop any others that might exist, then analyze and warm up.
			dropAllSoftDeleteIndexes()
			_, err := testDB.DB.Exec(ctx, "CREATE INDEX IF NOT EXISTS deployments_deleted ON deployments (deleted)")
			require.NoError(b, err)
			_, err = testDB.DB.Exec(ctx, "ANALYZE deployments")
			require.NoError(b, err)
			warmupQueries()
			runBenchmarks(b, "DeletedOnly")

			// --- Index configuration: state only ---
			dropAllSoftDeleteIndexes()
			_, err = testDB.DB.Exec(ctx, "CREATE INDEX deployments_state ON deployments (state)")
			require.NoError(b, err)
			_, err = testDB.DB.Exec(ctx, "ANALYZE deployments")
			require.NoError(b, err)
			warmupQueries()
			runBenchmarks(b, "StateOnly")

			// --- Index configuration: composite (state, deleted) ---
			dropAllSoftDeleteIndexes()
			_, err = testDB.DB.Exec(ctx, "CREATE INDEX deployments_state_deleted ON deployments (state, deleted)")
			require.NoError(b, err)
			_, err = testDB.DB.Exec(ctx, "ANALYZE deployments")
			require.NoError(b, err)
			warmupQueries()
			runBenchmarks(b, "Composite")

			// --- Index configuration: no index ---
			dropAllSoftDeleteIndexes()
			_, err = testDB.DB.Exec(ctx, "ANALYZE deployments")
			require.NoError(b, err)
			warmupQueries()
			runBenchmarks(b, "NoIndex")
		})
	}
}
