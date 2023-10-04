package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// Store is the interface to the events data layer.
//
//go:generate mockgen-wrapper
type Store interface {
	Get(ctx context.Context, id string) (*storage.AdministrationEvent, bool, error)
	GetByQuery(ctx context.Context, query *v1.Query) ([]*storage.AdministrationEvent, error)
	UpsertMany(ctx context.Context, objs []*storage.AdministrationEvent) error
	DeleteMany(ctx context.Context, identifiers []string) error
	GetMany(ctx context.Context, identifiers []string) ([]*storage.AdministrationEvent, []int, error)
}
