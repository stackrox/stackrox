package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	storage "github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Store defines the interface for Risk storage
//
//go:generate mockgen-wrapper
type Store interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Get(ctx context.Context, id string) (*storage.Risk, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.Risk, []int, error)
	Walk(context.Context, func(risk *storage.Risk) error) error
	Upsert(ctx context.Context, risk *storage.Risk) error
	Delete(ctx context.Context, id ...string) error
}
