package datastore

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

//go:generate mockgen-wrapper
type DataStore interface {
	CountVirtualMachines(ctx context.Context, query *v1.Query) (int, error)
	GetVirtualMachine(ctx context.Context, id string) (*storage.VirtualMachine, bool, error)
	UpsertVirtualMachine(ctx context.Context, virtualMachine *storage.VirtualMachine) error
	UpdateVirtualMachineScan(ctx context.Context, vmID string, scan *storage.VirtualMachineScan) error
	DeleteVirtualMachines(ctx context.Context, ids ...string) error
	Exists(ctx context.Context, id string) (bool, error)
	SearchRawVirtualMachines(ctx context.Context, query *v1.Query) ([]*storage.VirtualMachine, error)
}
