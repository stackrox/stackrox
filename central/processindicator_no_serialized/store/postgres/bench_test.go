//go:build sql_integration

package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	serializedStore "github.com/stackrox/rox/central/processindicator/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protocompat"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
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
		pgSearch.NoSerializedStoreOpts[storeType]{
			ChildFetcher: FetchChildren,
		},
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

// BenchmarkWriteStrategiesDetailed compares three write strategies at 1000 objects:
//   - PerRow:  current approach — one INSERT per parent + one INSERT per child + one DELETE per parent (4000 batch entries)
//   - Unnest:  bulk approach — one unnest INSERT for all parents + DELETE children + one unnest INSERT for all children (3 statements)
//   - UnnestParentOnly: like Unnest but only for parents (children still per-row), to isolate parent vs child contribution
func BenchmarkWriteStrategiesDetailed(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())
	db := pgtest.ForT(b)

	const batchSize = 1000

	// -- Strategy 1: current per-row INSERT ON CONFLICT batch --
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

	// -- Strategy 2: unnest for BOTH parent and child tables --
	b.Run("Unnest_1000", func(b *testing.B) {
		_ = New(db.DB) // ensure table exists
		for b.Loop() {
			b.StopTimer()
			objs := make([]*storage.ProcessIndicatorNoSerialized, batchSize)
			for i := range objs {
				objs[i] = makeNoSerializedIndicator(uuid.NewV4().String())
			}

			// Build parent arrays (one slice per column)
			pIDs := make([]string, 0, batchSize)
			pDeploymentIDs := make([]string, 0, batchSize)
			pContainerNames := make([]string, 0, batchSize)
			pPodIDs := make([]string, 0, batchSize)
			pPodUIDs := make([]string, 0, batchSize)
			pSignalIDs := make([]string, 0, batchSize)
			pSignalContainerIDs := make([]string, 0, batchSize)
			pSignalTimes := make([]*time.Time, 0, batchSize)
			pSignalNames := make([]string, 0, batchSize)
			pSignalArgs := make([]string, 0, batchSize)
			pSignalExecPaths := make([]string, 0, batchSize)
			pSignalPids := make([]int64, 0, batchSize)
			pSignalUIDs := make([]int64, 0, batchSize)
			pSignalGids := make([]int64, 0, batchSize)
			pSignalScrapeds := make([]bool, 0, batchSize)
			pClusterIDs := make([]string, 0, batchSize)
			pNamespaces := make([]string, 0, batchSize)
			pContainerStartTimes := make([]*time.Time, 0, batchSize)
			pImageIDs := make([]string, 0, batchSize)

			// Build child arrays
			cParentIDs := make([]string, 0, batchSize*2)
			cIdxs := make([]int32, 0, batchSize*2)
			cParentUIDs := make([]int64, 0, batchSize*2)
			cParentExecPaths := make([]string, 0, batchSize*2)

			for _, obj := range objs {
				pIDs = append(pIDs, obj.GetId())
				pDeploymentIDs = append(pDeploymentIDs, obj.GetDeploymentId())
				pContainerNames = append(pContainerNames, obj.GetContainerName())
				pPodIDs = append(pPodIDs, obj.GetPodId())
				pPodUIDs = append(pPodUIDs, obj.GetPodUid())
				pSignalIDs = append(pSignalIDs, obj.GetSignal().GetId())
				pSignalContainerIDs = append(pSignalContainerIDs, obj.GetSignal().GetContainerId())
				pSignalTimes = append(pSignalTimes, protocompat.NilOrTime(obj.GetSignal().GetTime()))
				pSignalNames = append(pSignalNames, obj.GetSignal().GetName())
				pSignalArgs = append(pSignalArgs, obj.GetSignal().GetArgs())
				pSignalExecPaths = append(pSignalExecPaths, obj.GetSignal().GetExecFilePath())
				pSignalPids = append(pSignalPids, int64(obj.GetSignal().GetPid()))
				pSignalUIDs = append(pSignalUIDs, int64(obj.GetSignal().GetUid()))
				pSignalGids = append(pSignalGids, int64(obj.GetSignal().GetGid()))
				pSignalScrapeds = append(pSignalScrapeds, obj.GetSignal().GetScraped())
				pClusterIDs = append(pClusterIDs, obj.GetClusterId())
				pNamespaces = append(pNamespaces, obj.GetNamespace())
				pContainerStartTimes = append(pContainerStartTimes, protocompat.NilOrTime(obj.GetContainerStartTime()))
				pImageIDs = append(pImageIDs, obj.GetImageId())

				for idx, li := range obj.GetSignal().GetLineageInfo() {
					cParentIDs = append(cParentIDs, obj.GetId())
					cIdxs = append(cIdxs, int32(idx))
					cParentUIDs = append(cParentUIDs, int64(li.GetParentUid()))
					cParentExecPaths = append(cParentExecPaths, li.GetParentExecFilePath())
				}
			}
			b.StartTimer()

			batch := &pgx.Batch{}

			// 1. Bulk upsert parents via unnest (3 statements total vs 4000 per-row)
			batch.Queue(`INSERT INTO process_indicator_no_serializeds
				(Id, DeploymentId, ContainerName, PodId, PodUid,
				 Signal_Id, Signal_ContainerId, Signal_Time, Signal_Name, Signal_Args,
				 Signal_ExecFilePath, Signal_Pid, Signal_Uid, Signal_Gid, Signal_Scraped,
				 ClusterId, Namespace, ContainerStartTime, ImageId)
				SELECT * FROM unnest(
					$1::uuid[], $2::uuid[], $3::text[], $4::text[], $5::uuid[],
					$6::text[], $7::text[], $8::timestamp[], $9::text[], $10::text[],
					$11::text[], $12::bigint[], $13::bigint[], $14::bigint[], $15::bool[],
					$16::uuid[], $17::text[], $18::timestamp[], $19::text[]
				)
				ON CONFLICT(Id) DO UPDATE SET
					DeploymentId = EXCLUDED.DeploymentId, ContainerName = EXCLUDED.ContainerName,
					PodId = EXCLUDED.PodId, PodUid = EXCLUDED.PodUid,
					Signal_Id = EXCLUDED.Signal_Id, Signal_ContainerId = EXCLUDED.Signal_ContainerId,
					Signal_Time = EXCLUDED.Signal_Time, Signal_Name = EXCLUDED.Signal_Name,
					Signal_Args = EXCLUDED.Signal_Args, Signal_ExecFilePath = EXCLUDED.Signal_ExecFilePath,
					Signal_Pid = EXCLUDED.Signal_Pid, Signal_Uid = EXCLUDED.Signal_Uid,
					Signal_Gid = EXCLUDED.Signal_Gid, Signal_Scraped = EXCLUDED.Signal_Scraped,
					ClusterId = EXCLUDED.ClusterId, Namespace = EXCLUDED.Namespace,
					ContainerStartTime = EXCLUDED.ContainerStartTime, ImageId = EXCLUDED.ImageId`,
				pIDs, pDeploymentIDs, pContainerNames, pPodIDs, pPodUIDs,
				pSignalIDs, pSignalContainerIDs, pSignalTimes, pSignalNames, pSignalArgs,
				pSignalExecPaths, pSignalPids, pSignalUIDs, pSignalGids, pSignalScrapeds,
				pClusterIDs, pNamespaces, pContainerStartTimes, pImageIDs,
			)

			// 2. Delete all existing children for these parents
			batch.Queue(`DELETE FROM process_indicator_no_serializeds_lineage_infos
				WHERE process_indicator_no_serializeds_Id = ANY($1::uuid[])`, pIDs)

			// 3. Bulk insert all children via unnest
			if len(cParentIDs) > 0 {
				batch.Queue(`INSERT INTO process_indicator_no_serializeds_lineage_infos
					(process_indicator_no_serializeds_Id, idx, ParentUid, ParentExecFilePath)
					SELECT * FROM unnest($1::uuid[], $2::int[], $3::bigint[], $4::text[])`,
					cParentIDs, cIdxs, cParentUIDs, cParentExecPaths,
				)
			}

			conn, err := db.DB.Acquire(ctx)
			require.NoError(b, err)
			batchResults := conn.SendBatch(ctx, batch)
			require.NoError(b, batchResults.Close())
			conn.Release()
		}
	})

	// -- Strategy 3: unnest parents only, per-row children (isolate contribution) --
	b.Run("UnnestParentsOnly_1000", func(b *testing.B) {
		_ = New(db.DB)
		for b.Loop() {
			b.StopTimer()
			objs := make([]*storage.ProcessIndicatorNoSerialized, batchSize)
			for i := range objs {
				objs[i] = makeNoSerializedIndicator(uuid.NewV4().String())
			}

			pIDs := make([]string, 0, batchSize)
			pDeploymentIDs := make([]string, 0, batchSize)
			pContainerNames := make([]string, 0, batchSize)
			pPodIDs := make([]string, 0, batchSize)
			pPodUIDs := make([]string, 0, batchSize)
			pSignalIDs := make([]string, 0, batchSize)
			pSignalContainerIDs := make([]string, 0, batchSize)
			pSignalTimes := make([]*time.Time, 0, batchSize)
			pSignalNames := make([]string, 0, batchSize)
			pSignalArgs := make([]string, 0, batchSize)
			pSignalExecPaths := make([]string, 0, batchSize)
			pSignalPids := make([]int64, 0, batchSize)
			pSignalUIDs := make([]int64, 0, batchSize)
			pSignalGids := make([]int64, 0, batchSize)
			pSignalScrapeds := make([]bool, 0, batchSize)
			pClusterIDs := make([]string, 0, batchSize)
			pNamespaces := make([]string, 0, batchSize)
			pContainerStartTimes := make([]*time.Time, 0, batchSize)
			pImageIDs := make([]string, 0, batchSize)

			for _, obj := range objs {
				pIDs = append(pIDs, obj.GetId())
				pDeploymentIDs = append(pDeploymentIDs, obj.GetDeploymentId())
				pContainerNames = append(pContainerNames, obj.GetContainerName())
				pPodIDs = append(pPodIDs, obj.GetPodId())
				pPodUIDs = append(pPodUIDs, obj.GetPodUid())
				pSignalIDs = append(pSignalIDs, obj.GetSignal().GetId())
				pSignalContainerIDs = append(pSignalContainerIDs, obj.GetSignal().GetContainerId())
				pSignalTimes = append(pSignalTimes, protocompat.NilOrTime(obj.GetSignal().GetTime()))
				pSignalNames = append(pSignalNames, obj.GetSignal().GetName())
				pSignalArgs = append(pSignalArgs, obj.GetSignal().GetArgs())
				pSignalExecPaths = append(pSignalExecPaths, obj.GetSignal().GetExecFilePath())
				pSignalPids = append(pSignalPids, int64(obj.GetSignal().GetPid()))
				pSignalUIDs = append(pSignalUIDs, int64(obj.GetSignal().GetUid()))
				pSignalGids = append(pSignalGids, int64(obj.GetSignal().GetGid()))
				pSignalScrapeds = append(pSignalScrapeds, obj.GetSignal().GetScraped())
				pClusterIDs = append(pClusterIDs, obj.GetClusterId())
				pNamespaces = append(pNamespaces, obj.GetNamespace())
				pContainerStartTimes = append(pContainerStartTimes, protocompat.NilOrTime(obj.GetContainerStartTime()))
				pImageIDs = append(pImageIDs, obj.GetImageId())
			}
			b.StartTimer()

			batch := &pgx.Batch{}

			// Unnest parents
			batch.Queue(`INSERT INTO process_indicator_no_serializeds
				(Id, DeploymentId, ContainerName, PodId, PodUid,
				 Signal_Id, Signal_ContainerId, Signal_Time, Signal_Name, Signal_Args,
				 Signal_ExecFilePath, Signal_Pid, Signal_Uid, Signal_Gid, Signal_Scraped,
				 ClusterId, Namespace, ContainerStartTime, ImageId)
				SELECT * FROM unnest(
					$1::uuid[], $2::uuid[], $3::text[], $4::text[], $5::uuid[],
					$6::text[], $7::text[], $8::timestamp[], $9::text[], $10::text[],
					$11::text[], $12::bigint[], $13::bigint[], $14::bigint[], $15::bool[],
					$16::uuid[], $17::text[], $18::timestamp[], $19::text[]
				)
				ON CONFLICT(Id) DO UPDATE SET
					DeploymentId = EXCLUDED.DeploymentId, ContainerName = EXCLUDED.ContainerName,
					PodId = EXCLUDED.PodId, PodUid = EXCLUDED.PodUid,
					Signal_Id = EXCLUDED.Signal_Id, Signal_ContainerId = EXCLUDED.Signal_ContainerId,
					Signal_Time = EXCLUDED.Signal_Time, Signal_Name = EXCLUDED.Signal_Name,
					Signal_Args = EXCLUDED.Signal_Args, Signal_ExecFilePath = EXCLUDED.Signal_ExecFilePath,
					Signal_Pid = EXCLUDED.Signal_Pid, Signal_Uid = EXCLUDED.Signal_Uid,
					Signal_Gid = EXCLUDED.Signal_Gid, Signal_Scraped = EXCLUDED.Signal_Scraped,
					ClusterId = EXCLUDED.ClusterId, Namespace = EXCLUDED.Namespace,
					ContainerStartTime = EXCLUDED.ContainerStartTime, ImageId = EXCLUDED.ImageId`,
				pIDs, pDeploymentIDs, pContainerNames, pPodIDs, pPodUIDs,
				pSignalIDs, pSignalContainerIDs, pSignalTimes, pSignalNames, pSignalArgs,
				pSignalExecPaths, pSignalPids, pSignalUIDs, pSignalGids, pSignalScrapeds,
				pClusterIDs, pNamespaces, pContainerStartTimes, pImageIDs,
			)

			// Per-row children (current approach)
			for _, obj := range objs {
				for childIdx, child := range obj.GetSignal().GetLineageInfo() {
					batch.Queue(`INSERT INTO process_indicator_no_serializeds_lineage_infos
						(process_indicator_no_serializeds_Id, idx, ParentUid, ParentExecFilePath)
						VALUES($1, $2, $3, $4)
						ON CONFLICT(process_indicator_no_serializeds_Id, idx) DO UPDATE SET
						process_indicator_no_serializeds_Id = EXCLUDED.process_indicator_no_serializeds_Id,
						idx = EXCLUDED.idx, ParentUid = EXCLUDED.ParentUid,
						ParentExecFilePath = EXCLUDED.ParentExecFilePath`,
						obj.GetId(), childIdx, child.GetParentUid(), child.GetParentExecFilePath(),
					)
				}
				batch.Queue(`DELETE FROM process_indicator_no_serializeds_lineage_infos
					WHERE process_indicator_no_serializeds_Id = $1 AND idx >= $2`,
					obj.GetId(), len(obj.GetSignal().GetLineageInfo()),
				)
			}

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

// BenchmarkGetSingleReadPathBreakdown isolates the two components of the
// NoSerialized read: parent column scan vs child table fetch.
func BenchmarkGetSingleReadPathBreakdown(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())
	db := pgtest.ForT(b)

	const seedCount = 1000
	noSerializedIDs := make([]string, seedCount)
	nSt := New(db.DB)
	// Also create a store without child fetcher to isolate parent-only reads
	nStNoChildren := pgSearch.NewNoSerializedStore[storeType](
		db.DB,
		schema,
		pkGetter,
		insertIntoProcessIndicatorNoSerializeds,
		nil,
		scanRow,
		scanRows,
		nil, nil, nil,
		targetResource,
	)

	nObjs := make([]*storage.ProcessIndicatorNoSerialized, seedCount)
	for i := 0; i < seedCount; i++ {
		id := uuid.NewV4().String()
		noSerializedIDs[i] = id
		nObjs[i] = makeNoSerializedIndicator(id)
	}
	require.NoError(b, nSt.UpsertMany(ctx, nObjs))

	b.Run("ParentOnly_NoChildFetch", func(b *testing.B) {
		i := 0
		for b.Loop() {
			id := noSerializedIDs[i%seedCount]
			obj, exists, err := nStNoChildren.Get(ctx, id)
			require.NoError(b, err)
			require.True(b, exists)
			require.NotNil(b, obj)
			i++
		}
	})

	b.Run("WithChildFetch", func(b *testing.B) {
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

// BenchmarkGetManyReadPathBreakdown isolates parent scan vs child fetch for batch reads.
func BenchmarkGetManyReadPathBreakdown(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())
	db := pgtest.ForT(b)

	const seedCount = 1000
	noSerializedIDs := make([]string, seedCount)
	nSt := New(db.DB)
	nStNoChildren := pgSearch.NewNoSerializedStore[storeType](
		db.DB,
		schema,
		pkGetter,
		insertIntoProcessIndicatorNoSerializeds,
		nil,
		scanRow,
		scanRows,
		nil, nil, nil,
		targetResource,
	)

	nObjs := make([]*storage.ProcessIndicatorNoSerialized, seedCount)
	for i := 0; i < seedCount; i++ {
		id := uuid.NewV4().String()
		noSerializedIDs[i] = id
		nObjs[i] = makeNoSerializedIndicator(id)
	}
	require.NoError(b, nSt.UpsertMany(ctx, nObjs))

	for _, batchSize := range []int{10, 100, 500} {
		b.Run(fmt.Sprintf("BatchSize_%d", batchSize), func(b *testing.B) {
			b.Run("ParentOnly_NoChildFetch", func(b *testing.B) {
				for b.Loop() {
					objs, missing, err := nStNoChildren.GetMany(ctx, noSerializedIDs[:batchSize])
					require.NoError(b, err)
					require.Empty(b, missing)
					require.Len(b, objs, batchSize)
				}
			})

			b.Run("WithChildFetch", func(b *testing.B) {
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
