package datastore

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	SearchVirtualMachines(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawVirtualMachines(ctx context.Context, q *v1.Query) ([]*storage.VirtualMachine, error)

	CountVirtualMachines(ctx context.Context) (int, error)
	GetVirtualMachine(ctx context.Context, sha string) (*storage.VirtualMachine, bool, error)

	UpsertVirtualMachine(ctx context.Context, virtualMachine *storage.VirtualMachine) error

	DeleteVirtualMachines(ctx context.Context, ids ...string) error
	Exists(ctx context.Context, id string) (bool, error)
}
