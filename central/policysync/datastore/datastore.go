package datastore

import (
	"context"

	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/policysync/datastore/store"
	pgStore "github.com/stackrox/rox/central/policysync/datastore/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	_ DataStore = (*datastoreImpl)(nil)

	once sync.Once
	d    DataStore

	log = logging.LoggerForModule()
)

// DataStore is the datastore for policy sync.
type DataStore interface {
	GetPolicySync(ctx context.Context) (*storage.PolicySync, bool, error)
	UpsertPolicySync(ctx context.Context, sync *storage.PolicySync) error
}

// Singleton provides the singleton.
func Singleton() DataStore {
	once.Do(func() {
		d = newDatastore(pgStore.New(globaldb.GetPostgres()))
	})
	return d
}

func newDatastore(store store.Store) DataStore {
	return &datastoreImpl{
		store: store,
	}
}

type datastoreImpl struct {
	store store.Store
}

func (d *datastoreImpl) GetPolicySync(ctx context.Context) (*storage.PolicySync, bool, error) {
	return d.store.Get(ctx)
}

func (d *datastoreImpl) UpsertPolicySync(ctx context.Context, sync *storage.PolicySync) error {
	log.Infof("Upserting policy sync %+v", sync)
	return d.store.Upsert(ctx, sync)
}
