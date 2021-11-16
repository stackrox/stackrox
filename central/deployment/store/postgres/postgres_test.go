package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/jackc/pgx/v4/pgxpool"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
)

func TestFlattened(t *testing.T) {
	search.Walk(v1.SearchCategory_DEPLOYMENTS, "deployment", (*storage.Deployment)(nil))
}

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
	db := setup()

	store := New(db)
	err := store.Upsert(fixtures.GetDeployment())
	if err != nil {
		panic(err)
	}
}

const num = 10000

func BenchmarkNormalizedTable(b *testing.B) {
	db := setup()

	store := New(db)
	err := store.Upsert(fixtures.GetDeployment())
	if err != nil {
		panic(err)
	}

	defer db.Close()

	deployments := make([]*storage.Deployment, 0, num)
	for i := 0; i < num; i++ {
		dep := fixtures.GetDeployment()
		dep.Id = uuid.NewV4().String()
		deployments = append(deployments, dep)
	}

	conn, err := db.Acquire(context.Background())
	if err != nil {
		panic(err)
	}
	conn.Release()

	t := time.Now()
	for _, dep := range deployments {
		if err := store.Upsert(dep); err != nil {
			panic(err)
		}
	}
	ms := time.Since(t).Milliseconds()
	log.Infof("Took ms: %d for %d (%0.4f average)", ms, num, float64(ms)/num)
}

func BenchmarkComposite(b *testing.B) {
	db := setup()

	createTable := "create table if not exists  deployment_composite ( id varchar primary key, json jsonb );"
	_, err := db.Exec(context.Background(), createTable)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	deployments := make([]*storage.Deployment, 0, num)
	for i := 0; i < num; i++ {
		dep := fixtures.GetDeployment()
		dep.Id = uuid.NewV4().String()
		deployments = append(deployments, dep)
	}

	conn, err := db.Acquire(context.Background())
	if err != nil {
		panic(err)
	}
	conn.Release()

	m := jsonpb.Marshaler{}
	insert := "insert into deployment_composite (id, json) values($1, $2);"
	t := time.Now()
	for _, dep := range deployments {
		s, err := m.MarshalToString(dep)
		if err != nil {
			panic(err)
		}
		_, err = db.Exec(context.Background(), insert, dep.GetId(), s)
		if err != nil {
			panic(err)
		}
	}
	ms := time.Since(t).Milliseconds()
	log.Infof("Took ms: %d for %d (%0.4f average)", ms, num, float64(ms)/num)
}
