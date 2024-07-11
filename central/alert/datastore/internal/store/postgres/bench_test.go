package postgres

import (
	"context"
	"fmt"
	"testing"

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
		idx = append(idx, a.Id)
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
			err := store.UpsertMany(ctx, alerts[:n])
			if err != nil {
				b.Fatal(err)
			}
		})
		b.Run(fmt.Sprintf("get %d alerts", n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _, err := store.GetMany(ctx, idx[:n])
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
