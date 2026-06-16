//go:build sql_integration

package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func setupBench(b *testing.B) (Store, context.Context) {
	b.Helper()
	testDB := pgtest.ForT(b)
	store := New(testDB.DB)
	ctx := sac.WithAllAccess(context.Background())
	return store, ctx
}

func BenchmarkUpsert(b *testing.B) {
	store, ctx := setupBench(b)

	obj := makeBenchObj("bench")
	b.ResetTimer()
	for b.Loop() {
		b.StopTimer()
		obj.Id = uuid.NewV4().String()
		b.StartTimer()

		if err := store.Upsert(ctx, obj); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUpsertMany(b *testing.B) {
	for _, size := range []int{10, 100, 500} {
		b.Run(fmt.Sprintf("batch_%d", size), func(b *testing.B) {
			store, ctx := setupBench(b)

			objs := make([]*storage.TestNoSerialized, size)
			for i := range objs {
				objs[i] = makeBenchObj(fmt.Sprintf("bench-%d", i))
			}

			b.ResetTimer()
			for b.Loop() {
				b.StopTimer()
				for _, obj := range objs {
					obj.Id = uuid.NewV4().String()
				}
				b.StartTimer()

				if err := store.UpsertMany(ctx, objs); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkGet(b *testing.B) {
	store, ctx := setupBench(b)

	obj := makeBenchObj("bench-get")
	if err := store.Upsert(ctx, obj); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for b.Loop() {
		if _, _, err := store.Get(ctx, obj.GetId()); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWalk(b *testing.B) {
	store, ctx := setupBench(b)

	for i := 0; i < 1000; i++ {
		obj := makeBenchObj(fmt.Sprintf("walk-%d", i))
		if err := store.Upsert(ctx, obj); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for b.Loop() {
		_ = store.Walk(ctx, func(_ *storage.TestNoSerialized) error {
			return nil
		})
	}
}

func makeBenchObj(name string) *storage.TestNoSerialized {
	now := time.Now().Truncate(time.Microsecond)
	return &storage.TestNoSerialized{
		Id:          uuid.NewV4().String(),
		Name:        name,
		Description: "benchmark " + name,
		Int32Val:    42,
		Int64Val:    9999999999,
		Uint64Val:   200,
		BoolVal:     true,
		FloatVal:    3.14,
		DoubleVal:   2.71828,
		Priority:    storage.TestNoSerialized_HIGH_PRIORITY,
		CreatedAt:   timestamppb.New(now),
		ClusterId:   uuid.NewV4().String(),
		Tags:        []string{"tag1", "tag2", "tag3"},
		Metadata: &storage.TestNoSerialized_Metadata{
			Author:   "bench-author",
			Version:  "1.0.0",
			Revision: 7,
		},
	}
}
