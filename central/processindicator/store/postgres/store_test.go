package postgres

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	_ "github.com/lib/pq"
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

	// tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource
	// ids := pgx.Identifier([]string{"processindicator"})
	//vals, err := pool.CopyFrom(context.Background(), ids, []string{"id", "deploymentid", "serialized"}, pgx.CopyFromRows([][]interface{}{
	//	{
	//		"id1",
	//		"dep1",
	//		[]byte("{}"),
	//	},
	//	{
	//		"id2",
	//		"dep2",
	//		[]byte("{}"),
	//	},
	//}))
	//if err != nil {
	//	panic(err)
	//}

	var values []string
	var data []interface{}
	for i := 0; i < 100000; i++ {
		values = append(values, fmt.Sprintf("($%d, $%d, $%d)", i*3+1, i*3+2, i*3+3))
		data = append(data, fmt.Sprintf("%d", i+1), "dep", []byte("{}"))
	}
	line := fmt.Sprintf("insert into processindicator(id, deploymentid, serialized) Values%s", strings.Join(values, ", "))
	_, err := pool.Exec(context.Background(), line, data...)
	if err != nil {
		panic(err)
	}

	fmt.Println(err)
}
