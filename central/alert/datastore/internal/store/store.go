package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Store provides storage functionality for alerts.
//
//go:generate mockgen-wrapper
type Store interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Walk(ctx context.Context, fn func(*storage.Alert) error) error
	WalkByQuery(ctx context.Context, q *v1.Query, fn func(*storage.Alert) error) error
	GetIDs(ctx context.Context) ([]string, error)
	Get(ctx context.Context, id string) (*storage.Alert, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.Alert, []int, error)
	GetByQuery(ctx context.Context, query *v1.Query) ([]*storage.Alert, error)
	Upsert(ctx context.Context, alert *storage.Alert) error
	UpsertMany(ctx context.Context, alerts []*storage.Alert) error
	Delete(ctx context.Context, id string) error
	DeleteMany(ctx context.Context, ids []string) error
}
