package postgres

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
)

func BenchmarkMany(b *testing.B) {
	var alerts []*storage.Alert
	const alertsNum = 10000
	for i := 0; i < alertsNum; i++ {
		alert := &storage.Alert{}
		err := testutils.FullInit(alert, testutils.UniqueInitializer(), testutils.JSONFieldsFilter)
		if err != nil {
			b.Fatal(err)
		}
		alerts = append(alerts, alert)
	}

	var idx []string
	for _, a := range alerts {
		idx = append(idx, a.GetId())
	}

	testDB := pgtest.ForT(b)
	store := New(testDB.DB)

	ctx := sac.WithAllAccess(context.Background())
	err := store.UpsertMany(ctx, alerts)
	if err != nil {
		b.Fatal(err)
	}

	for n := 1; n < alertsNum; n = n * 2 {
		b.Run(fmt.Sprintf("upsert %d alerts", n), func(b *testing.B) {
			var maxHeap uint64
			var maxHeapObj uint64
			ticker := time.NewTicker(10 * time.Millisecond)
			go func() {
				var m runtime.MemStats
				for range ticker.C {
					runtime.GC()
					runtime.ReadMemStats(&m)
					maxHeap = max(maxHeap, m.HeapAlloc)
					maxHeapObj = max(maxHeap, m.HeapObjects)
				}
			}()

			for b.Loop() {
				err := store.UpsertMany(ctx, alerts[:n])
				if err != nil {
					b.Fatal(err)
				}
			}
			ticker.Stop()
			b.ReportMetric(float64(maxHeap), "max_heap_bytes")
			b.ReportMetric(float64(maxHeapObj), "max_heap_objects")
		})
		b.Run(fmt.Sprintf("get %d alerts", n), func(b *testing.B) {
			for b.Loop() {
				_, _, err := store.GetMany(ctx, idx[:n])
				if err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run(fmt.Sprintf("walk %d alerts", n), func(b *testing.B) {
			for b.Loop() {
				count := 0
				err := store.Walk(ctx, func(obj *storeType) error {
					count++
					return nil
				})
				if err != nil {
					b.Fatal(err)
				}
				if alertsNum != count {
					b.Fatalf("Expected %d alerts, got %d", alertsNum, count)
				}
			}
		})
	}
}
