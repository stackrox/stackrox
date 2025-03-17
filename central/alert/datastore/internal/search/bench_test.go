package search

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/central/alert/datastore/internal/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
)

func BenchmarkSearchListAlerts(b *testing.B) {
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

	store := postgres.New(testDB)
	searcher := New(store)

	ctx := sac.WithAllAccess(context.Background())
	err := store.UpsertMany(ctx, alerts)
	if err != nil {
		b.Fatal(err)
	}

	for n := 1; n < alertsNum; n = n * 2 {
		b.Run(fmt.Sprintf("search %d alerts", n), func(b *testing.B) {
			q := &v1.Query{Pagination: &v1.QueryPagination{
				Limit: int32(n),
			}}

			for i := 0; i < b.N; i++ {
				_, err := searcher.SearchListAlerts(ctx, q, false)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
