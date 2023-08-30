package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// Store is the interface to the notifications data layer.
//
//go:generate mockgen-wrapper
type Store interface {
	Get(ctx context.Context, id string) (*storage.Notification, bool, error)
	GetByQuery(ctx context.Context, query *v1.Query) ([]*storage.Notification, error)
	UpsertMany(ctx context.Context, objs []*storage.Notification) error
}
