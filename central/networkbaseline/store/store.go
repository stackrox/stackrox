package store

import (
	"context"

	"github.com/stackrox/rox/central/networkbaseline/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	pgPkg "github.com/stackrox/rox/pkg/postgres"
)

// Store provides storage functionality for network baselines.
//
//go:generate mockgen-wrapper
type Store interface {
	Exists(ctx context.Context, id string) (bool, error)

	GetIDs(ctx context.Context) ([]string, error)
	Get(ctx context.Context, id string) (*storage.NetworkBaseline, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.NetworkBaseline, []int, error)

	Upsert(ctx context.Context, baseline *storage.NetworkBaseline) error
	UpsertMany(ctx context.Context, baselines []*storage.NetworkBaseline) error
	Delete(ctx context.Context, id string) error
	DeleteMany(ctx context.Context, ids []string) error

	Walk(ctx context.Context, fn func(baseline *storage.NetworkBaseline) error) error

	Begin(ctx context.Context) (context.Context, *pgPkg.Tx, error)
}

// storeWrapper is a wrapper around the generated store which also exposes transaction functionality.
// The reason for requiring this is that we also have an in-memory store for the auth machine to machine config,
// and require rollbacks for upsert and delete.
type storeWrapper struct {
	db pgPkg.DB
	postgres.Store
}

// New creates a new store.
func New(db pgPkg.DB) Store {
	return &storeWrapper{
		db:    db,
		Store: postgres.New(db),
	}
}

// Begin begins a transaction, returning the transaction context from the given context and the transaction itself.
func (s *storeWrapper) Begin(ctx context.Context) (context.Context, *pgPkg.Tx, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	return pgPkg.ContextWithTx(ctx, tx), tx, nil
}
