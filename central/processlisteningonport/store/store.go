package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality.
//
//go:generate mockgen-wrapper
type Store interface {
	Upsert(ctx context.Context, obj *storage.ProcessListeningOnPortStorage) error
	UpsertMany(ctx context.Context, objs []*storage.ProcessListeningOnPortStorage) error
	Delete(ctx context.Context, id string) error
	DeleteByQuery(ctx context.Context, q *v1.Query) ([]string, error)
	DeleteMany(ctx context.Context, identifiers []string) error

	Count(ctx context.Context) (int, error)
	Exists(ctx context.Context, id string) (bool, error)

	Get(ctx context.Context, id string) (*storage.ProcessListeningOnPortStorage, bool, error)
	GetByQuery(ctx context.Context, query *v1.Query) ([]*storage.ProcessListeningOnPortStorage, error)
	GetMany(ctx context.Context, identifiers []string) ([]*storage.ProcessListeningOnPortStorage, []int, error)
	GetIDs(ctx context.Context) ([]string, error)

	Walk(ctx context.Context, fn func(obj *storage.ProcessListeningOnPortStorage) error) error

	GetProcessListeningOnPort(
		ctx context.Context,
		deploymentID string,
	) ([]*storage.ProcessListeningOnPort, error)
}
