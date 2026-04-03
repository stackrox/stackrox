//go:build sql_integration

package postgres

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	serializedStore "github.com/stackrox/rox/central/processindicator/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
)

func makeSerializedIndicator(id string) *storage.ProcessIndicator {
	return &storage.ProcessIndicator{
		Id:            id,
		DeploymentId:  uuid.NewV4().String(),
		ContainerName: "container-" + id[:8],
		PodId:         "pod-" + id[:8],
		PodUid:        uuid.NewV4().String(),
		ClusterId:     uuid.NewV4().String(),
		Namespace:     "namespace-bench",
		Signal: &storage.ProcessSignal{
			ContainerId:  "cid-" + id[:8],
			Name:         "apt-get",
			Args:         "install nmap",
			ExecFilePath: "/usr/bin/apt-get",
			Pid:          1234,
			Uid:          1000,
			Gid:          1000,
			Scraped:      false,
			LineageInfo: []*storage.ProcessSignal_LineageInfo{
				{ParentUid: 22, ParentExecFilePath: "/bin/bash"},
				{ParentUid: 1, ParentExecFilePath: "/sbin/init"},
			},
		},
	}
}

func makeNoSerializedIndicator(id string) *storage.ProcessIndicatorNoSerialized {
	return &storage.ProcessIndicatorNoSerialized{
		Id:            id,
		DeploymentId:  uuid.NewV4().String(),
		ContainerName: "container-" + id[:8],
		PodId:         "pod-" + id[:8],
		PodUid:        uuid.NewV4().String(),
		ClusterId:     uuid.NewV4().String(),
		Namespace:     "namespace-bench",
		Signal: &storage.ProcessSignalNoSerialized{
			ContainerId:  "cid-" + id[:8],
			Name:         "apt-get",
			Args:         "install nmap",
			ExecFilePath: "/usr/bin/apt-get",
			Pid:          1234,
			Uid:          1000,
			Gid:          1000,
			Scraped:      false,
			LineageInfo: []*storage.ProcessSignalNoSerialized_LineageInfoNoSerialized{
				{ParentUid: 22, ParentExecFilePath: "/bin/bash"},
				{ParentUid: 1, ParentExecFilePath: "/sbin/init"},
			},
		},
	}
}

// newNoSerializedStoreWithCopyFrom creates a NoSerialized store with copyFrom
// ENABLED, to benchmark the per-parent COPY FROM path vs INSERT batch.
func newNoSerializedStoreWithCopyFrom(db pgtest.TestPostgres) Store {
	return pgSearch.NewNoSerializedStore[storeType](
		db.DB,
		schema,
		pkGetter,
		insertIntoProcessIndicatorNoSerializeds,
		copyFromProcessIndicatorNoSerializeds,
		scanRow,
		scanRows,
		nil,
		nil,
		nil,
		targetResource,
	)
}

// BenchmarkUpsertSingle compares single-object Upsert performance.
func BenchmarkUpsertSingle(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())
	db := pgtest.ForT(b)

	b.Run("Serialized", func(b *testing.B) {
		store := serializedStore.New(db.DB)
		for b.Loop() {
			obj := makeSerializedIndicator(uuid.NewV4().String())
			require.NoError(b, store.Upsert(ctx, obj))
		}
	})

	b.Run("NoSerialized", func(b *testing.B) {
		store := New(db.DB)
		for b.Loop() {
			obj := makeNoSerializedIndicator(uuid.NewV4().String())
			require.NoError(b, store.Upsert(ctx, obj))
		}
	})
}

// BenchmarkUpsertMany compares batch UpsertMany performance at various batch sizes.
// Tests three write strategies:
//   - Serialized:          standard serialized store (INSERT ON CONFLICT < 100, COPY FROM >= 100)
//   - NoSerialized:        NoSerialized with COPY FROM for large batches (current default)
//   - NoSerialized_NoCopy: NoSerialized with INSERT ON CONFLICT for ALL batch sizes
func BenchmarkUpsertMany(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())

	for _, batchSize := range []int{10, 100, 500, 1000} {
		b.Run(fmt.Sprintf("BatchSize_%d", batchSize), func(b *testing.B) {
			db := pgtest.ForT(b)

			b.Run("Serialized", func(b *testing.B) {
				store := serializedStore.New(db.DB)
				for b.Loop() {
					b.StopTimer()
					objs := make([]*storage.ProcessIndicator, batchSize)
					for i := range objs {
						objs[i] = makeSerializedIndicator(uuid.NewV4().String())
					}
					b.StartTimer()
					require.NoError(b, store.UpsertMany(ctx, objs))
				}
			})

			b.Run("NoSerialized_CopyFrom", func(b *testing.B) {
				store := newNoSerializedStoreWithCopyFrom(*db)
				for b.Loop() {
					b.StopTimer()
					objs := make([]*storage.ProcessIndicatorNoSerialized, batchSize)
					for i := range objs {
						objs[i] = makeNoSerializedIndicator(uuid.NewV4().String())
					}
					b.StartTimer()
					require.NoError(b, store.UpsertMany(ctx, objs))
				}
			})

			b.Run("NoSerialized_InsertBatch", func(b *testing.B) {
				store := New(db.DB) // default: INSERT ON CONFLICT batch (no copyFrom)
				for b.Loop() {
					b.StopTimer()
					objs := make([]*storage.ProcessIndicatorNoSerialized, batchSize)
					for i := range objs {
						objs[i] = makeNoSerializedIndicator(uuid.NewV4().String())
					}
					b.StartTimer()
					require.NoError(b, store.UpsertMany(ctx, objs))
				}
			})
		})
	}
}

// BenchmarkWriteStrategiesDetailed compares write strategies at 1000 objects:
//   - PerRow:        one INSERT per parent (1000 batch entries)
//   - BulkUnnest:    generated bulkInsert with unnest (1 unnest + N per-row for non-unnestable)
func BenchmarkWriteStrategiesDetailed(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())
	db := pgtest.ForT(b)

	const batchSize = 1000

	// -- Strategy 1: per-row INSERT ON CONFLICT batch --
	b.Run("PerRowBatch_1000", func(b *testing.B) {
		_ = New(db.DB) // ensure table exists
		for b.Loop() {
			b.StopTimer()
			objs := make([]*storage.ProcessIndicatorNoSerialized, batchSize)
			for i := range objs {
				objs[i] = makeNoSerializedIndicator(uuid.NewV4().String())
			}
			batch := &pgx.Batch{}
			for _, obj := range objs {
				if err := insertIntoProcessIndicatorNoSerializeds(batch, obj); err != nil {
					b.Fatal(err)
				}
			}
			b.StartTimer()

			conn, err := db.DB.Acquire(ctx)
			require.NoError(b, err)
			batchResults := conn.SendBatch(ctx, batch)
			require.NoError(b, batchResults.Close())
			conn.Release()
		}
	})

	// -- Strategy 2: generated bulk unnest --
	b.Run("BulkUnnest_1000", func(b *testing.B) {
		_ = New(db.DB) // ensure table exists
		for b.Loop() {
			b.StopTimer()
			objs := make([]*storage.ProcessIndicatorNoSerialized, batchSize)
			for i := range objs {
				objs[i] = makeNoSerializedIndicator(uuid.NewV4().String())
			}
			batch := &pgx.Batch{}
			if err := bulkInsertIntoProcessIndicatorNoSerializeds(batch, objs); err != nil {
				b.Fatal(err)
			}
			b.StartTimer()

			conn, err := db.DB.Acquire(ctx)
			require.NoError(b, err)
			batchResults := conn.SendBatch(ctx, batch)
			require.NoError(b, batchResults.Close())
			conn.Release()
		}
	})
}

// BenchmarkGetSingle compares single-object Get performance.
func BenchmarkGetSingle(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())
	db := pgtest.ForT(b)

	const seedCount = 1000
	serializedIDs := make([]string, seedCount)
	noSerializedIDs := make([]string, seedCount)

	sSt := serializedStore.New(db.DB)
	nSt := New(db.DB)

	sObjs := make([]*storage.ProcessIndicator, seedCount)
	nObjs := make([]*storage.ProcessIndicatorNoSerialized, seedCount)
	for i := 0; i < seedCount; i++ {
		id := uuid.NewV4().String()
		serializedIDs[i] = id
		sObjs[i] = makeSerializedIndicator(id)

		id2 := uuid.NewV4().String()
		noSerializedIDs[i] = id2
		nObjs[i] = makeNoSerializedIndicator(id2)
	}
	require.NoError(b, sSt.UpsertMany(ctx, sObjs))
	require.NoError(b, nSt.UpsertMany(ctx, nObjs))

	b.Run("Serialized", func(b *testing.B) {
		i := 0
		for b.Loop() {
			id := serializedIDs[i%seedCount]
			obj, exists, err := sSt.Get(ctx, id)
			require.NoError(b, err)
			require.True(b, exists)
			require.NotNil(b, obj)
			i++
		}
	})

	b.Run("NoSerialized", func(b *testing.B) {
		i := 0
		for b.Loop() {
			id := noSerializedIDs[i%seedCount]
			obj, exists, err := nSt.Get(ctx, id)
			require.NoError(b, err)
			require.True(b, exists)
			require.NotNil(b, obj)
			i++
		}
	})
}

// BenchmarkGetSingleReadPathBreakdown benchmarks single-object Get
// with inlined LineageInfo (no child table fetch needed).
func BenchmarkGetSingleReadPathBreakdown(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())
	db := pgtest.ForT(b)

	const seedCount = 1000
	noSerializedIDs := make([]string, seedCount)
	nSt := New(db.DB)

	nObjs := make([]*storage.ProcessIndicatorNoSerialized, seedCount)
	for i := 0; i < seedCount; i++ {
		id := uuid.NewV4().String()
		noSerializedIDs[i] = id
		nObjs[i] = makeNoSerializedIndicator(id)
	}
	require.NoError(b, nSt.UpsertMany(ctx, nObjs))

	b.Run("InlinedLineageInfo", func(b *testing.B) {
		i := 0
		for b.Loop() {
			id := noSerializedIDs[i%seedCount]
			obj, exists, err := nSt.Get(ctx, id)
			require.NoError(b, err)
			require.True(b, exists)
			require.NotNil(b, obj)
			require.NotEmpty(b, obj.GetSignal().GetLineageInfo())
			i++
		}
	})
}

// BenchmarkGetManyReadPathBreakdown benchmarks batch GetMany
// with inlined LineageInfo (no child table fetch needed).
func BenchmarkGetManyReadPathBreakdown(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())
	db := pgtest.ForT(b)

	const seedCount = 1000
	noSerializedIDs := make([]string, seedCount)
	nSt := New(db.DB)

	nObjs := make([]*storage.ProcessIndicatorNoSerialized, seedCount)
	for i := 0; i < seedCount; i++ {
		id := uuid.NewV4().String()
		noSerializedIDs[i] = id
		nObjs[i] = makeNoSerializedIndicator(id)
	}
	require.NoError(b, nSt.UpsertMany(ctx, nObjs))

	for _, batchSize := range []int{10, 100, 500} {
		b.Run(fmt.Sprintf("BatchSize_%d", batchSize), func(b *testing.B) {
			for b.Loop() {
				objs, missing, err := nSt.GetMany(ctx, noSerializedIDs[:batchSize])
				require.NoError(b, err)
				require.Empty(b, missing)
				require.Len(b, objs, batchSize)
			}
		})
	}
}

// BenchmarkGetMany compares batch GetMany performance.
func BenchmarkGetMany(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())
	db := pgtest.ForT(b)

	const seedCount = 1000
	serializedIDs := make([]string, seedCount)
	noSerializedIDs := make([]string, seedCount)

	sSt := serializedStore.New(db.DB)
	nSt := New(db.DB)

	sObjs := make([]*storage.ProcessIndicator, seedCount)
	nObjs := make([]*storage.ProcessIndicatorNoSerialized, seedCount)
	for i := 0; i < seedCount; i++ {
		id := uuid.NewV4().String()
		serializedIDs[i] = id
		sObjs[i] = makeSerializedIndicator(id)

		id2 := uuid.NewV4().String()
		noSerializedIDs[i] = id2
		nObjs[i] = makeNoSerializedIndicator(id2)
	}
	require.NoError(b, sSt.UpsertMany(ctx, sObjs))
	require.NoError(b, nSt.UpsertMany(ctx, nObjs))

	for _, batchSize := range []int{10, 100, 500} {
		b.Run(fmt.Sprintf("BatchSize_%d", batchSize), func(b *testing.B) {
			b.Run("Serialized", func(b *testing.B) {
				for b.Loop() {
					objs, missing, err := sSt.GetMany(ctx, serializedIDs[:batchSize])
					require.NoError(b, err)
					require.Empty(b, missing)
					require.Len(b, objs, batchSize)
				}
			})

			b.Run("NoSerialized", func(b *testing.B) {
				for b.Loop() {
					objs, missing, err := nSt.GetMany(ctx, noSerializedIDs[:batchSize])
					require.NoError(b, err)
					require.Empty(b, missing)
					require.Len(b, objs, batchSize)
				}
			})
		})
	}
}

// BenchmarkWalk compares full-table Walk performance.
func BenchmarkWalk(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())
	db := pgtest.ForT(b)

	const seedCount = 2000

	sSt := serializedStore.New(db.DB)
	nSt := New(db.DB)

	sObjs := make([]*storage.ProcessIndicator, seedCount)
	nObjs := make([]*storage.ProcessIndicatorNoSerialized, seedCount)
	for i := 0; i < seedCount; i++ {
		sObjs[i] = makeSerializedIndicator(uuid.NewV4().String())
		nObjs[i] = makeNoSerializedIndicator(uuid.NewV4().String())
	}
	require.NoError(b, sSt.UpsertMany(ctx, sObjs))
	require.NoError(b, nSt.UpsertMany(ctx, nObjs))

	b.Run("Serialized", func(b *testing.B) {
		for b.Loop() {
			count := 0
			err := sSt.Walk(ctx, func(_ *storage.ProcessIndicator) error {
				count++
				return nil
			})
			require.NoError(b, err)
			require.Equal(b, seedCount, count)
		}
	})

	b.Run("NoSerialized", func(b *testing.B) {
		for b.Loop() {
			count := 0
			err := nSt.Walk(ctx, func(_ *storage.ProcessIndicatorNoSerialized) error {
				count++
				return nil
			})
			require.NoError(b, err)
			require.Equal(b, seedCount, count)
		}
	})
}

// BenchmarkCount compares Count performance (should be identical since
// both use the same SQL path, included as a baseline/sanity check).
func BenchmarkCount(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())
	db := pgtest.ForT(b)

	const seedCount = 1000

	sSt := serializedStore.New(db.DB)
	nSt := New(db.DB)

	sObjs := make([]*storage.ProcessIndicator, seedCount)
	nObjs := make([]*storage.ProcessIndicatorNoSerialized, seedCount)
	for i := 0; i < seedCount; i++ {
		sObjs[i] = makeSerializedIndicator(uuid.NewV4().String())
		nObjs[i] = makeNoSerializedIndicator(uuid.NewV4().String())
	}
	require.NoError(b, sSt.UpsertMany(ctx, sObjs))
	require.NoError(b, nSt.UpsertMany(ctx, nObjs))

	b.Run("Serialized", func(b *testing.B) {
		for b.Loop() {
			count, err := sSt.Count(ctx, search.EmptyQuery())
			require.NoError(b, err)
			require.Equal(b, seedCount, count)
		}
	})

	b.Run("NoSerialized", func(b *testing.B) {
		for b.Loop() {
			count, err := nSt.Count(ctx, search.EmptyQuery())
			require.NoError(b, err)
			require.Equal(b, seedCount, count)
		}
	})
}
