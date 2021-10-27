package postgres

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"

	_ "github.com/lib/pq"
)

func TestStore(t *testing.T) {
	source := "host=localhost port=5432 user=postgres sslmode=disable statement_timeout=60000"
	db, err := sql.Open("postgres", source)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		panic(err)
	}

	type Object struct {
		Name string
		Key string
		Value string
	}

	objs := []Object {
		{
			Name: "name1",
			Key: "key1",
			Value: "value1",
		},
		{
			Name: "name2",
			Key: "key2",
			Value: "value2",
		},
	}

	_, err = db.Exec("create table if not exists testing (id varchar, objs jsonb)")
	if err != nil {
		panic(err)
	}

	data, err := json.Marshal(objs)
	if err != nil {
		panic(err)
	}

	_, err = db.Exec("insert into testing(id, objs) values($1, $2)", "id", string(data))
	if err != nil {
		panic(err)
	}

	return


	store := New(db)

	risk, exists, err := store.Get("511ce712-45db-493d-9580-7c93931159a1")
	fmt.Println(risk, exists, err)
}
