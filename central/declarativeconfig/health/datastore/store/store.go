package store

import (
	"context"

	"github.com/stackrox/rox/central/declarativeconfig/health/datastore/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	pgPkg "github.com/stackrox/rox/pkg/postgres"
)

// Store is the interface to the config health data layer
type Store interface {
	Get(ctx context.Context, id string) (*storage.DeclarativeConfigHealth, bool, error)
	Upsert(ctx context.Context, obj *storage.DeclarativeConfigHealth) error
	UpsertMany(ctx context.Context, objs []*storage.DeclarativeConfigHealth) error
	Delete(ctx context.Context, id string) error
	Walk(ctx context.Context, fn func(obj *storage.DeclarativeConfigHealth) error) error

	// Begin starts a transaction and returns a context with the transaction
	Begin(ctx context.Context) (context.Context, *pgPkg.Tx, error)
}

// storeWrapper wraps the generated postgres store to add transaction support
type storeWrapper struct {
	db pgPkg.DB
	postgres.Store
}

// New returns a new Store instance with transaction support
func New(db pgPkg.DB) Store {
	return &storeWrapper{
		db:    db,
		Store: postgres.New(db),
	}
}

// Begin starts a new database transaction
func (s *storeWrapper) Begin(ctx context.Context) (context.Context, *pgPkg.Tx, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	return pgPkg.ContextWithTx(ctx, tx), tx, nil
}
