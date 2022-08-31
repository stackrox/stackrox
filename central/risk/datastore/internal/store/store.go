package store

import (
	"context"

	storage "github.com/stackrox/rox/generated/storage"
)

// Store defines the interface for Risk storage
//
//go:generate mockgen-wrapper
type Store interface {
	Get(ctx context.Context, id string) (*storage.Risk, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.Risk, []int, error)
	Walk(context.Context, func(risk *storage.Risk) error) error
	Upsert(ctx context.Context, risk *storage.Risk) error
	Delete(ctx context.Context, id string) error
}
