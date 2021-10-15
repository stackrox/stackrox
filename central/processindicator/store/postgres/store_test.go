package postgres

import (
	"database/sql"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stackrox/rox/generated/storage"
)

func TestT(t *testing.T) {
	source := "host=localhost port=5432 user=postgres sslmode=disable statement_timeout=60000"
	db, err := sql.Open("postgres", source)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	store := New(db)

	err = store.UpsertMany([]*storage.ProcessIndicator{
		{
			Id: "1",
			Namespace: "stackrox",
		},
		{
			Id: "2",
			Namespace: "stackrox2",
		},
	})
	if err != nil {
		panic(err)
	}

}