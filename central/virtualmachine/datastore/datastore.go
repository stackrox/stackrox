package datastore

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

//go:generate mockgen-wrapper
type DataStore interface {
	CountVirtualMachines(ctx context.Context) (int, error)
	GetVirtualMachine(ctx context.Context, sha string) (*storage.VirtualMachine, bool, error)
	GetAllVirtualMachines(ctx context.Context) ([]*storage.VirtualMachine, error)
	CreateVirtualMachine(ctx context.Context, virtualMachine *storage.VirtualMachine) error
	UpsertVirtualMachine(ctx context.Context, virtualMachine *storage.VirtualMachine) error
	DeleteVirtualMachines(ctx context.Context, ids ...string) error
	Exists(ctx context.Context, id string) (bool, error)
}
