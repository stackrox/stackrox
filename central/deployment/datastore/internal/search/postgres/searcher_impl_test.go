package search

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/lib/pq"
	deploymentPGStore "github.com/stackrox/rox/central/deployment/store/postgres"
	"github.com/stackrox/rox/pkg/search"
)

func TestT(t *testing.T) {
	source := "host=localhost port=5432 user=postgres sslmode=disable statement_timeout=60000"
	db, err := sql.Open("postgres", source)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	searcher := New(db, deploymentPGStore.NewFullStore(db))

	results, err := searcher.Search(context.Background(), search.NewQueryBuilder().ProtoQuery())
	if err != nil {
		panic(err)
	}
	fmt.Printf("Results: %d\n", len(results))
}
