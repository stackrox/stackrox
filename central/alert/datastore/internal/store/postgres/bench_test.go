package postgres

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
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
	b.Cleanup(func() {
		testDB.Teardown(b)
	})
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
		b.Run(fmt.Sprintf("walk %d alerts", n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				count := 0
				err := store.Walk(ctx, func(obj *storeType) error {
					count++
					return nil
				})
				assert.NoError(b, err)
				assert.Equal(b, alertsNum, count)
			}
		})
	}
}
