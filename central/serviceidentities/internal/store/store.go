package store

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for service identities.
//
//go:generate mockgen-wrapper
type Store interface {
	GetAll(ctx context.Context) ([]*storage.ServiceIdentity, error)
	Upsert(ctx context.Context, obj *storage.ServiceIdentity) error
}
