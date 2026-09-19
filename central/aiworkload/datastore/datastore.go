package datastore

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

//go:generate mockgen-wrapper
type DataStore interface {
	CountAIWorkloads(ctx context.Context, query *v1.Query) (int, error)
	GetAIWorkload(ctx context.Context, id string) (*storage.AIWorkload, bool, error)
	UpsertAIWorkload(ctx context.Context, workload *storage.AIWorkload) error
	DeleteAIWorkloads(ctx context.Context, ids ...string) error
	Exists(ctx context.Context, id string) (bool, error)
	SearchRawAIWorkloads(ctx context.Context, query *v1.Query) ([]*storage.AIWorkload, error)
	Walk(ctx context.Context, fn func(w *storage.AIWorkload) error) error
}
