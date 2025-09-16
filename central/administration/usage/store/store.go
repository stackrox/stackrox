package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// Store interface provides methods to access a persistent storage.
//
//go:generate mockgen-wrapper
type Store interface {
	Upsert(ctx context.Context, obj *storage.SecuredUnits) error
	WalkByQuery(ctx context.Context, query *v1.Query, fn func(obj *storage.SecuredUnits) error) error
}
