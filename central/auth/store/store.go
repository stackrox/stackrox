package store

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store is the interface to the auth machine to machine data layer.
type Store interface {
	Get(ctx context.Context, id string) (*storage.AuthMachineToMachineConfig, bool, error)
	Upsert(ctx context.Context, obj *storage.AuthMachineToMachineConfig) error
	Delete(ctx context.Context, id string) error
	GetAll(ctx context.Context) ([]*storage.AuthMachineToMachineConfig, error)
}
