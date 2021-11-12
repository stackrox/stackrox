package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v4"
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

const num = 10000

func BenchmarkNormalizedTable(b *testing.B) {
	db := setup()

	createDeploymentTable := "create table if not exists deployment_normalized ( id varchar primary key );"
	_, err := db.Exec(context.Background(), createDeploymentTable)
	if err != nil {
		panic(err)
	}

	createContainerTable := "create table if not exists container_normalized ( parent_deployment_id varchar, container_idx integer, name varchar, primary key(parent_deployment_id, container_idx), CONSTRAINT fk_parent_deployment_id FOREIGN KEY (parent_deployment_id) REFERENCES deployment_normalized(id) )"
	_, err = db.Exec(context.Background(), createContainerTable)
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
	defer conn.Release()

	t := time.Now()
	for _, dep := range deployments {
		tx, err := conn.BeginTx(context.Background(), pgx.TxOptions{})
		if err != nil {
			panic(err)
		}
		_, err = tx.Exec(context.Background(), "insert into deployment_normalized (id) values($1) on conflict do nothing", dep.GetId())
		if err != nil {
			panic(err)
		}
		//createContainerTable := "create table if not exists container_normalized ( parent_deployment_id varchar, container_idx integer, name varchar, PRIMARY KEY(parent_deployment_id, container_idx))"

		for idx, c := range dep.GetContainers() {
			_, err = tx.Exec(context.Background(), "insert into container_normalized (parent_deployment_id, container_idx, name) values($1, $2, $3) on conflict do nothing", dep.GetId(), idx, c.GetName())
			if err != nil {
				panic(err)
			}
		}
		_, err = tx.Exec(context.Background(), "delete from container_normalized where parent_deployment_id = $1 and container_idx >= $2", dep.GetId(), len(dep.GetContainers()))
		if err != nil {
			panic(err)
		}

		err = tx.Commit(context.Background())
		if err != nil {
			panic(err)
		}
	}
	ms := time.Since(t).Milliseconds()
	log.Infof("Took ms: %d for %d (%0.4f average)", ms, num, float64(ms)/num)
}

func BenchmarkComposite(b *testing.B) {
	db := setup()

	createTable := "create table if not exists  deployment_composite ( id varchar primary key, containers deployment_container[] );"
	_, err := db.Exec(context.Background(), createTable)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	insert := "insert into deployment_composite (id, containers) values($1, ARRAY[($2, $3)::deployment_container, ($4, $5)::deployment_container]);"
	for i := 0; i < num; i++ {
		a := strconv.Itoa(i*2)
		b := strconv.Itoa(i*2+1)
		_, err := db.Exec(context.Background(), insert, "dep-id" + a, a, a, b, b)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkJsonb(b *testing.B) {
	db := setup()

	createTable := "create table if not exists deployment_jsonb ( id varchar primary key, containers jsonb );"
	_, err := db.Exec(context.Background(), createTable)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	type value struct {
		Id, Name string
	}

	insert := "insert into deployment_jsonb (id, containers) values($1, $2);"
	for i := 0; i < num; i++ {
		a := strconv.Itoa(i*2)
		b := strconv.Itoa(i*2+1)
		values := []value {
			{
				Id: a,
				Name: a,
			},
			{
				Id: b,
				Name: b,
			},
		}
		bytes, err := json.Marshal(values)
		if err != nil {
			panic(err)
		}

		_, err = db.Exec(context.Background(), insert, "dep-id" +a, bytes)
		if err != nil {
			panic(err)
		}
	}
}
