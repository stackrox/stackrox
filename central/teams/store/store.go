package store

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

type Store interface {
	Get(ctx context.Context, id string) (*storage.Team, bool, error)
	Upsert(ctx context.Context, obj *storage.Team) error
	GetAll(ctx context.Context) ([]*storage.Team, error)
	Walk(ctx context.Context, fn func(obj *storage.Team) error) error
}
