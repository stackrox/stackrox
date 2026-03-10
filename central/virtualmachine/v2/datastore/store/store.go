package store

import (
	"context"

	"github.com/stackrox/rox/central/virtualmachine/v2/datastore/store/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Store provides storage functionality for VirtualMachineV2 and its related
// scan, component, and CVE data.
//
//go:generate mockgen-wrapper
type Store interface {
	// UpsertVM upserts a VM. Hash-comparison determines whether a full write
	// or timestamp-only update is performed.
	UpsertVM(ctx context.Context, vm *storage.VirtualMachineV2) error

	// UpsertScan upserts scan data (scan, components, CVEs) for a VM.
	// Hash-comparison determines whether a full replace or scan_time-only
	// update is performed. CVE created_at timestamps are preserved across
	// delete/re-insert cycles.
	UpsertScan(ctx context.Context, vmID string, parts common.VMScanParts) error

	// Delete removes a VM and all associated data (FK cascade).
	Delete(ctx context.Context, id string) error
	// DeleteMany removes multiple VMs and all associated data.
	DeleteMany(ctx context.Context, ids []string) error

	// Count returns the number of VMs matching the query.
	Count(ctx context.Context, q *v1.Query) (int, error)
	// Search returns search results matching the query.
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	// Get returns the VM with the given ID.
	Get(ctx context.Context, id string) (*storage.VirtualMachineV2, bool, error)
	// GetMany returns the VMs with the given IDs.
	GetMany(ctx context.Context, ids []string) ([]*storage.VirtualMachineV2, []int, error)

	// Walk iterates over all VMs, calling fn for each.
	Walk(ctx context.Context, fn func(vm *storage.VirtualMachineV2) error) error
	// WalkByQuery iterates over VMs matching the query, calling fn for each.
	WalkByQuery(ctx context.Context, q *v1.Query, fn func(vm *storage.VirtualMachineV2) error) error
}
