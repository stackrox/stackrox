package datastore

import (
	"context"

	"github.com/stackrox/rox/central/virtualmachine/v2/datastore/store/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// DataStore is the public interface for the VM v2 datastore.
//
//go:generate mockgen-wrapper
type DataStore interface {
	// CountVirtualMachines returns the number of VMs matching the query.
	CountVirtualMachines(ctx context.Context, query *v1.Query) (int, error)

	// GetVirtualMachine returns the VM with the given ID.
	GetVirtualMachine(ctx context.Context, id string) (*storage.VirtualMachineV2, bool, error)
	// GetManyVirtualMachines returns the VMs with the given IDs.
	GetManyVirtualMachines(ctx context.Context, ids []string) ([]*storage.VirtualMachineV2, []int, error)

	// UpsertVirtualMachine upserts a VM. The store performs hash-based change
	// detection to avoid unnecessary writes.
	UpsertVirtualMachine(ctx context.Context, vm *storage.VirtualMachineV2) error

	// UpsertScan upserts scan data (scan, components, CVEs) for a VM.
	// Hash-based change detection avoids unnecessary writes. CVE created_at
	// timestamps are preserved across scan replacements.
	UpsertScan(ctx context.Context, vmID string, parts common.VMScanParts) error

	// DeleteVirtualMachines removes VMs and all associated data (FK cascade).
	DeleteVirtualMachines(ctx context.Context, ids ...string) error

	// Exists returns whether a VM with the given ID exists.
	Exists(ctx context.Context, id string) (bool, error)

	// Search returns search results matching the query.
	Search(ctx context.Context, query *v1.Query) ([]search.Result, error)

	// SearchRawVirtualMachines returns VMs matching the query.
	SearchRawVirtualMachines(ctx context.Context, query *v1.Query) ([]*storage.VirtualMachineV2, error)

	// Walk iterates over all VMs, calling fn for each.
	Walk(ctx context.Context, fn func(vm *storage.VirtualMachineV2) error) error
}
