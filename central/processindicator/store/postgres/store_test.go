package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	_ "github.com/lib/pq"
	"github.com/stackrox/rox/generated/storage"
)

func setup() *pgxpool.Pool {
	config, err := pgxpool.ParseConfig("database=postgres pool_min_conns=100 pool_max_conns=100 host=localhost port=5432 user=connorgorman sslmode=disable statement_timeout=60000")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", config)

	db, err := pgxpool.ConnectConfig(context.Background(), config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	return db
}

func TestT(t *testing.T) {
	pool := setup()

	store := New(pool)

	err := store.UpsertMany([]*storage.ProcessIndicator{
		{
			Id:        "1",
			Namespace: "stackrox",
		},
		{
			Id:        "2",
			Namespace: "stackrox2",
		},
	})
	if err != nil {
		panic(err)
	}
}
