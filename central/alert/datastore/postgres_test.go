package datastore

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	_ "github.com/lib/pq"
	alertPGIndex "github.com/stackrox/rox/central/alert/datastore/internal/index/postgres"
	alertPGStore "github.com/stackrox/rox/central/alert/datastore/internal/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

/*
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}

*/

func TestPGX(t *testing.T) {
	conn, err := pgx.Connect(context.Background(), "host=localhost port=5432 user=postgres sslmode=disable statement_timeout=60000")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	if err := conn.Ping(context.Background()); err != nil {
		panic(err)
	}
}

func TestT(t *testing.T) {
	config, err := pgxpool.ParseConfig("pool_min_conns=100 pool_max_conns=100 host=localhost port=5432 user=postgres sslmode=disable statement_timeout=60000")
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

	alertStore := alertPGStore.New(db)
	fmt.Println(alertStore)

	alertIndex := alertPGIndex.NewIndexer(db)
	fmt.Println(alertIndex)

	qb := search.NewQueryBuilder().
		AddStrings(
			search.ViolationState,
			storage.ViolationState_ACTIVE.String(),
			storage.ViolationState_ATTEMPTED.String()).
		AddStrings(search.Cluster, "remote").
		AddStringsHighlighted(search.ClusterID, search.WildcardString)

	pq := qb.ProtoQuery()

	pq.Pagination = &v1.QueryPagination{
		Limit: 10,
	}

	results, err := alertIndex.Search(pq, nil)
	if err != nil {
		panic(err)
	}
	for _, r := range results {
		fmt.Printf("result: %s - %+v\n", r.ID, r.Matches)
	}
}
