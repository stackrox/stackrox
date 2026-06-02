//go:build sql_integration

package postgres

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func makeBenchObj() *storage.TestNoSerializedObj {
	return &storage.TestNoSerializedObj{
		Id:          uuid.NewV4().String(),
		Name:        "bench-obj",
		ValueInt32:  42,
		ValueInt64:  123456789,
		ValueUint32: 100,
		ValueUint64: 9999999999,
		ValueBool:   true,
		ValueFloat:  3.14,
		ValueEnum:   storage.TestNoSerializedEnum_TEST_NO_SERIALIZED_ENUM_ACTIVE,
		CreatedAt:   timestamppb.Now(),
		Nested: &storage.TestNoSerializedNested{
			Label: "nested",
			Score: 999,
		},
		Tags: []string{"a", "b", "c"},
		Metadata: []*storage.TestNoSerializedMetadata{
			{Key: "k1", Value: "v1"},
			{Key: "k2", Value: "v2"},
		},
	}
}

func setupBench(b *testing.B) (Store, context.Context) {
	b.Helper()
	testDB := pgtest.ForT(b)
	ctx := sac.WithAllAccess(context.Background())
	store := CreateTableAndNewStore(ctx, testDB.DB, testDB.GetGormDB(b))
	return store, ctx
}

func BenchmarkUpsert(b *testing.B) {
	store, ctx := setupBench(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := store.Upsert(ctx, makeBenchObj()); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUpsertMany(b *testing.B) {
	for _, batchSize := range []int{10, 100, 500} {
		b.Run(fmt.Sprintf("batch_%d", batchSize), func(b *testing.B) {
			store, ctx := setupBench(b)

			objs := make([]*storage.TestNoSerializedObj, batchSize)
			for i := range objs {
				objs[i] = makeBenchObj()
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Re-generate IDs to avoid upsert-over-same-row effects
				for _, obj := range objs {
					obj.Id = uuid.NewV4().String()
				}
				if err := store.UpsertMany(ctx, objs); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkGet(b *testing.B) {
	store, ctx := setupBench(b)

	obj := makeBenchObj()
	if err := store.Upsert(ctx, obj); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := store.Get(ctx, obj.GetId()); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetMany(b *testing.B) {
	for _, batchSize := range []int{10, 100, 500} {
		b.Run(fmt.Sprintf("batch_%d", batchSize), func(b *testing.B) {
			store, ctx := setupBench(b)

			ids := make([]string, batchSize)
			for i := range ids {
				obj := makeBenchObj()
				ids[i] = obj.GetId()
				if err := store.Upsert(ctx, obj); err != nil {
					b.Fatal(err)
				}
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, _, err := store.GetMany(ctx, ids); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkCount(b *testing.B) {
	store, ctx := setupBench(b)

	for i := 0; i < 100; i++ {
		if err := store.Upsert(ctx, makeBenchObj()); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := store.Count(ctx, search.EmptyQuery()); err != nil {
			b.Fatal(err)
		}
	}
}
