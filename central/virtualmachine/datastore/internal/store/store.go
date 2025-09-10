package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// VirtualMachineStore provide the storage functionality for virtual machines
//
//go:generate mockgen-wrapper
type VirtualMachineStore interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
	Exists(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (*storage.VirtualMachine, bool, error)
	Walk(ctx context.Context, fn func(machine *storage.VirtualMachine) error) error

	GetByQueryFn(ctx context.Context, query *v1.Query, fn func(machine *storage.VirtualMachine) error) error

	DeleteMany(ctx context.Context, identifiers []string) error
	UpsertMany(ctx context.Context, objs []*storage.VirtualMachine) error
}
