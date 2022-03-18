package cluster

import (
	"context"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/cve/datastore"
	"github.com/stackrox/rox/central/cve/datastore/cluster/internal/search"
	"github.com/stackrox/rox/central/cve/datastore/cluster/internal/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	ds datastore.DataStore
)

func initialize() {
	var err error
	ds, err = newDatastore(context.TODO(), globaldb.GetPostgres())
	utils.CrashOnError(err)
}

func newDatastore(ctx context.Context, db *pgxpool.Pool) (*datastoreImpl, error) {
	storage := postgres.New(ctx, db)
	return &datastoreImpl{
		storage:  storage,
		searcher: search.New(storage, postgres.NewIndexer(db)),
	}, nil
}

// Singleton returns a singleton instance of node cve datastore
func Singleton() datastore.DataStore {
	once.Do(initialize)
	return ds
}
