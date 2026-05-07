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

func makeJsonbIndicator(id string) *storage.ProcessIndicatorJsonb {
	return &storage.ProcessIndicatorJsonb{
		Id:            id,
		DeploymentId:  uuid.NewV4().String(),
		ContainerName: "container-" + id[:8],
		PodId:         "pod-" + id[:8],
		PodUid:        uuid.NewV4().String(),
		ClusterId:     uuid.NewV4().String(),
		Namespace:     "namespace-bench",
		Signal: &storage.ProcessSignalJsonb{
			ContainerId:  "cid-" + id[:8],
			Name:         "apt-get",
			Args:         "install nmap",
			ExecFilePath: "/usr/bin/apt-get",
			Pid:          1234,
			Uid:          1000,
			Gid:          1000,
			Scraped:      false,
			LineageInfo: []*storage.ProcessSignalJsonb_LineageInfoJsonb{
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

// BenchmarkUpsertSingle compares single-object Upsert: bytea vs jsonb vs no-serialized.
func BenchmarkUpsertSingle(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())
	db := pgtest.ForT(b)

	b.Run("Serialized_bytea", func(b *testing.B) {
		store := serializedStore.New(db.DB)
		for b.Loop() {
			obj := makeSerializedIndicator(uuid.NewV4().String())
			require.NoError(b, store.Upsert(ctx, obj))
		}
	})

	b.Run("Jsonb", func(b *testing.B) {
		store := New(db.DB)
		for b.Loop() {
			obj := makeJsonbIndicator(uuid.NewV4().String())
			require.NoError(b, store.Upsert(ctx, obj))
		}
	})

	b.Run("NoSerialized", func(b *testing.B) {
		store := noSerializedStore.New(db.DB)
		for b.Loop() {
			obj := makeNoSerializedIndicator(uuid.NewV4().String())
			require.NoError(b, store.Upsert(ctx, obj))
		}
	})
}

// BenchmarkUpsertMany compares batch UpsertMany: bytea vs jsonb vs no-serialized.
func BenchmarkUpsertMany(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())

	for _, batchSize := range []int{10, 100, 500} {
		b.Run(fmt.Sprintf("BatchSize_%d", batchSize), func(b *testing.B) {
			db := pgtest.ForT(b)

			b.Run("Serialized_bytea", func(b *testing.B) {
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

			b.Run("Jsonb", func(b *testing.B) {
				store := New(db.DB)
				for b.Loop() {
					b.StopTimer()
					objs := make([]*storage.ProcessIndicatorJsonb, batchSize)
					for i := range objs {
						objs[i] = makeJsonbIndicator(uuid.NewV4().String())
					}
					b.StartTimer()
					require.NoError(b, store.UpsertMany(ctx, objs))
				}
			})

			b.Run("NoSerialized", func(b *testing.B) {
				store := noSerializedStore.New(db.DB)
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

// BenchmarkGetSingle compares single-object Get: bytea vs jsonb vs no-serialized.
func BenchmarkGetSingle(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())
	db := pgtest.ForT(b)

	const seedCount = 1000

	serializedIDs := make([]string, seedCount)
	jsonbIDs := make([]string, seedCount)
	noSerializedIDs := make([]string, seedCount)

	sSt := serializedStore.New(db.DB)
	jSt := New(db.DB)
	nSt := noSerializedStore.New(db.DB)

	sObjs := make([]*storage.ProcessIndicator, seedCount)
	jObjs := make([]*storage.ProcessIndicatorJsonb, seedCount)
	nObjs := make([]*storage.ProcessIndicatorNoSerialized, seedCount)
	for i := 0; i < seedCount; i++ {
		id := uuid.NewV4().String()
		serializedIDs[i] = id
		sObjs[i] = makeSerializedIndicator(id)

		id2 := uuid.NewV4().String()
		jsonbIDs[i] = id2
		jObjs[i] = makeJsonbIndicator(id2)

		id3 := uuid.NewV4().String()
		noSerializedIDs[i] = id3
		nObjs[i] = makeNoSerializedIndicator(id3)
	}
	require.NoError(b, sSt.UpsertMany(ctx, sObjs))
	require.NoError(b, jSt.UpsertMany(ctx, jObjs))
	require.NoError(b, nSt.UpsertMany(ctx, nObjs))

	b.Run("Serialized_bytea", func(b *testing.B) {
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

	b.Run("Jsonb", func(b *testing.B) {
		i := 0
		for b.Loop() {
			id := jsonbIDs[i%seedCount]
			obj, exists, err := jSt.Get(ctx, id)
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

// BenchmarkGetMany compares batch GetMany: bytea vs jsonb vs no-serialized.
func BenchmarkGetMany(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())
	db := pgtest.ForT(b)

	const seedCount = 1000

	serializedIDs := make([]string, seedCount)
	jsonbIDs := make([]string, seedCount)
	noSerializedIDs := make([]string, seedCount)

	sSt := serializedStore.New(db.DB)
	jSt := New(db.DB)
	nSt := noSerializedStore.New(db.DB)

	sObjs := make([]*storage.ProcessIndicator, seedCount)
	jObjs := make([]*storage.ProcessIndicatorJsonb, seedCount)
	nObjs := make([]*storage.ProcessIndicatorNoSerialized, seedCount)
	for i := 0; i < seedCount; i++ {
		id := uuid.NewV4().String()
		serializedIDs[i] = id
		sObjs[i] = makeSerializedIndicator(id)

		id2 := uuid.NewV4().String()
		jsonbIDs[i] = id2
		jObjs[i] = makeJsonbIndicator(id2)

		id3 := uuid.NewV4().String()
		noSerializedIDs[i] = id3
		nObjs[i] = makeNoSerializedIndicator(id3)
	}
	require.NoError(b, sSt.UpsertMany(ctx, sObjs))
	require.NoError(b, jSt.UpsertMany(ctx, jObjs))
	require.NoError(b, nSt.UpsertMany(ctx, nObjs))

	for _, batchSize := range []int{10, 100, 500} {
		b.Run(fmt.Sprintf("BatchSize_%d", batchSize), func(b *testing.B) {
			b.Run("Serialized_bytea", func(b *testing.B) {
				for b.Loop() {
					objs, missing, err := sSt.GetMany(ctx, serializedIDs[:batchSize])
					require.NoError(b, err)
					require.Empty(b, missing)
					require.Len(b, objs, batchSize)
				}
			})

			b.Run("Jsonb", func(b *testing.B) {
				for b.Loop() {
					objs, missing, err := jSt.GetMany(ctx, jsonbIDs[:batchSize])
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

// BenchmarkCount compares Count performance (baseline — should be identical).
func BenchmarkCount(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())
	db := pgtest.ForT(b)

	const seedCount = 1000

	sSt := serializedStore.New(db.DB)
	jSt := New(db.DB)
	nSt := noSerializedStore.New(db.DB)

	sObjs := make([]*storage.ProcessIndicator, seedCount)
	jObjs := make([]*storage.ProcessIndicatorJsonb, seedCount)
	nObjs := make([]*storage.ProcessIndicatorNoSerialized, seedCount)
	for i := 0; i < seedCount; i++ {
		sObjs[i] = makeSerializedIndicator(uuid.NewV4().String())
		jObjs[i] = makeJsonbIndicator(uuid.NewV4().String())
		nObjs[i] = makeNoSerializedIndicator(uuid.NewV4().String())
	}
	require.NoError(b, sSt.UpsertMany(ctx, sObjs))
	require.NoError(b, jSt.UpsertMany(ctx, jObjs))
	require.NoError(b, nSt.UpsertMany(ctx, nObjs))

	b.Run("Serialized_bytea", func(b *testing.B) {
		for b.Loop() {
			count, err := sSt.Count(ctx, search.EmptyQuery())
			require.NoError(b, err)
			require.Equal(b, seedCount, count)
		}
	})

	b.Run("Jsonb", func(b *testing.B) {
		for b.Loop() {
			count, err := jSt.Count(ctx, search.EmptyQuery())
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
