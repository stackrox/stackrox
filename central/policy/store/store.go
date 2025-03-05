package store

import (
	"context"

	policyPGStore "github.com/stackrox/rox/central/policy/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pgPkg "github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/search"
)

// Store provides storage functionality for policies.
//
//go:generate mockgen-wrapper
type Store interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Get(ctx context.Context, id string) (*storage.Policy, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.Policy, []int, error)
	GetAll(ctx context.Context) ([]*storage.Policy, error)
	GetIDs(ctx context.Context) ([]string, error)

	Upsert(ctx context.Context, obj *storage.Policy) error
	UpsertMany(ctx context.Context, objs []*storage.Policy) error

	Delete(ctx context.Context, id string) error
	DeleteMany(ctx context.Context, ids []string) error

	Begin(ctx context.Context) (context.Context, *pgPkg.Tx, error)
}

type storeWrapper struct {
	db    pgPkg.DB
	store policyPGStore.Store
}

func New(db pgPkg.DB) Store {
	return &storeWrapper{
		db:    db,
		store: policyPGStore.New(db),
	}
}

func (s *storeWrapper) Count(ctx context.Context, q *v1.Query) (int, error) {
	return s.store.Count(ctx, q)
}

func (s *storeWrapper) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return s.store.Search(ctx, q)
}

func (s *storeWrapper) Get(ctx context.Context, id string) (*storage.Policy, bool, error) {
	return s.store.Get(ctx, id)
}

func (s *storeWrapper) GetMany(ctx context.Context, ids []string) ([]*storage.Policy, []int, error) {
	return s.store.GetMany(ctx, ids)
}

func (s *storeWrapper) GetAll(ctx context.Context) ([]*storage.Policy, error) {
	return s.store.GetAll(ctx)
}

func (s *storeWrapper) GetIDs(ctx context.Context) ([]string, error) {
	return s.store.GetIDs(ctx)
}

func (s *storeWrapper) Upsert(ctx context.Context, obj *storage.Policy) error {
	return s.store.Upsert(ctx, obj)
}

func (s *storeWrapper) UpsertMany(ctx context.Context, objs []*storage.Policy) error {
	return s.store.UpsertMany(ctx, objs)
}

func (s *storeWrapper) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

func (s *storeWrapper) DeleteMany(ctx context.Context, ids []string) error {
	return s.store.DeleteMany(ctx, ids)
}

func (s *storeWrapper) Begin(ctx context.Context) (context.Context, *pgPkg.Tx, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	return pgPkg.ContextWithTx(ctx, tx), tx, nil
}
