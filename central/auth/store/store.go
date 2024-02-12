package store

import (
	"context"

	authPGStore "github.com/stackrox/rox/central/auth/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	pgPkg "github.com/stackrox/rox/pkg/postgres"
)

// Store is the interface to the auth machine to machine data layer.
type Store interface {
	Get(ctx context.Context, id string) (*storage.AuthMachineToMachineConfig, bool, error)
	Upsert(ctx context.Context, obj *storage.AuthMachineToMachineConfig) error
	Delete(ctx context.Context, id string) error
	GetAll(ctx context.Context) ([]*storage.AuthMachineToMachineConfig, error)
	Begin(ctx context.Context) (context.Context, *pgPkg.Tx, error)
}

// storeWrapper is a wrapper around the generated store which also exposes transaction functionality.
// The reason for requiring this is that we also have an in-memory store for the auth machine to machine config,
// and require rollbacks for upsert and delete.
type storeWrapper struct {
	db    pgPkg.DB
	store authPGStore.Store
}

// New creates a new store.
func New(db pgPkg.DB) Store {
	return &storeWrapper{
		db:    db,
		store: authPGStore.New(db),
	}
}

func (s *storeWrapper) Get(ctx context.Context, id string) (*storage.AuthMachineToMachineConfig, bool, error) {
	return s.store.Get(ctx, id)
}

func (s *storeWrapper) Upsert(ctx context.Context, obj *storage.AuthMachineToMachineConfig) error {
	return s.store.Upsert(ctx, obj)
}

func (s *storeWrapper) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

func (s *storeWrapper) GetAll(ctx context.Context) ([]*storage.AuthMachineToMachineConfig, error) {
	return s.store.GetAll(ctx)
}

// Begin begins a transaction, returning the transaction context from the given context and the transaction itself.
func (s *storeWrapper) Begin(ctx context.Context) (context.Context, *pgPkg.Tx, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	return pgPkg.ContextWithTx(ctx, tx), tx, nil
}
