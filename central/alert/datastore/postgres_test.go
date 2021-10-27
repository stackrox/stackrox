package datastore

import (
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/lib/pq"
	alertPGIndex "github.com/stackrox/rox/central/alert/datastore/internal/index/postgres"
	alertPGStore "github.com/stackrox/rox/central/alert/datastore/internal/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/search"
)

func TestT(t *testing.T) {
	source := "host=localhost port=5432 user=postgres sslmode=disable statement_timeout=60000"
	db, err := sql.Open("postgres", source)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	alertStore := alertPGStore.New(db)
	fmt.Println(alertStore)

	alertIndex := alertPGIndex.NewIndexer(db)
	fmt.Println(alertIndex)

	alert := fixtures.GetAlert()

	if err := alertStore.Upsert(alert); err != nil {
		panic(err)
	}

	qb := search.NewQueryBuilder().
		AddStrings(
			search.ViolationState,
			storage.ViolationState_ACTIVE.String(),
			storage.ViolationState_ATTEMPTED.String()).
		AddStrings(search.Cluster, "remote")

	pq := qb.ProtoQuery()
	pq.Pagination = &v1.QueryPagination{
		SortOptions: []*v1.QuerySortOption{
			{
				Field:    search.LifecycleStage.String(),
				Reversed: true,
			},
		},
	}

	results, err := alertIndex.Search(pq, nil)
	if err != nil {
		panic(err)
	}
	for _, r := range results {
		fmt.Println("result:", r.ID)
	}
}
