package datastore

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	pgIndex "github.com/stackrox/rox/central/risk/datastore/internal/index/postgres"
	pgSearch "github.com/stackrox/rox/central/risk/datastore/internal/search"
	pgStore "github.com/stackrox/rox/central/risk/datastore/internal/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
)

func TestRiskSearch(t *testing.T) {
	config, err := pgxpool.ParseConfig("pool_min_conns=100 pool_max_conns=100 host=localhost port=5432 database=postgres user=connorgorman sslmode=disable statement_timeout=60000")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", config)

	db, err := pgxpool.ConnectConfig(context.Background(), config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	store := pgStore.New(db)
	indexer := pgIndex.NewIndexer(db)
	searcher := pgSearch.New(store, indexer)

	results, err := searcher.Search(sac.WithAllAccess(context.Background()), search.NewQueryBuilder().AddExactMatches(search.RiskSubjectType, storage.RiskSubjectType_IMAGE.String()).ProtoQuery())
	if err != nil {
		panic(err)
	}
	fmt.Println(results)
}
