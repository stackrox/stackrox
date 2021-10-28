package datastore

import (
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/lib/pq"
	index "github.com/stackrox/rox/central/processindicator/index/postgres"
	store "github.com/stackrox/rox/central/processindicator/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/uuid"
)

func TestT(t *testing.T) {
	source := "host=localhost port=5432 user=postgres sslmode=disable statement_timeout=60000"
	db, err := sql.Open("postgres", source)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	processStore := store.New(db)
	fmt.Println(processStore)

	processIndex := index.NewIndexer(db)
	fmt.Println(processIndex)

	process1 := fixtures.GetProcessIndicator()
	process2 := fixtures.GetProcessIndicator()
	process2.Id = uuid.NewV4().String()

	if err := processStore.UpsertMany([]*storage.ProcessIndicator{process1, process2}); err != nil {
		panic(err)
	}
}
