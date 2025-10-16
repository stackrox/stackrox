package datastore

import (
	"context"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	// Use arbitrary reference time for deterministic tests (avoids flaky tests due to system clock changes)
	arbitraryNow     = time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	arbitraryNowFunc = func() time.Time { return arbitraryNow }
)

func TestVirtualMachineTelemetry(t *testing.T) {
	// Ensure feature flag is enabled for these tests
	t.Setenv(features.VirtualMachines.EnvVar(), "true")

	tests := map[string]struct {
		vms                            []*storage.VirtualMachine
		expectedClustersWithRunningVMs int
		expectedTotalVMs               int
		expectedVMsWithActiveAgents    int
	}{
		"should return zero for all metrics when no virtual machines exist": {
			vms:                            []*storage.VirtualMachine{},
			expectedClustersWithRunningVMs: 0,
			expectedTotalVMs:               0,
			expectedVMsWithActiveAgents:    0,
		},
		"should count single cluster with one running VM and recent scan": {
			vms: []*storage.VirtualMachine{
				storage.VirtualMachine_builder{Id: "vm1", ClusterId: "cluster1", Name: "test-vm", State: storage.VirtualMachine_RUNNING, Scan: createScanWithAge(1 * time.Hour)}.Build(),
			},
			expectedClustersWithRunningVMs: 1,
			expectedTotalVMs:               1,
			expectedVMsWithActiveAgents:    1,
		},
		"should count total VMs correctly including stopped ones": {
			vms: []*storage.VirtualMachine{
				storage.VirtualMachine_builder{Id: "vm1", ClusterId: "cluster1", Name: "vm-1", State: storage.VirtualMachine_RUNNING}.Build(),
				storage.VirtualMachine_builder{Id: "vm2", ClusterId: "cluster1", Name: "vm-2", State: storage.VirtualMachine_STOPPED}.Build(),
				storage.VirtualMachine_builder{Id: "vm3", ClusterId: "cluster2", Name: "vm-3", State: storage.VirtualMachine_UNKNOWN}.Build(),
			},
			expectedClustersWithRunningVMs: 1, // Only cluster1 has RUNNING VM
			expectedTotalVMs:               3, // All VMs counted
			expectedVMsWithActiveAgents:    0, // None have recent scan data
		},
		"should exclude VMs with old scans from active agent count": {
			vms: []*storage.VirtualMachine{
				storage.VirtualMachine_builder{Id: "vm1", ClusterId: "cluster1", Name: "vm-1", State: storage.VirtualMachine_RUNNING, Scan: createScanWithAge(1 * time.Hour)}.Build(),
				storage.VirtualMachine_builder{Id: "vm2", ClusterId: "cluster1", Name: "vm-2", State: storage.VirtualMachine_RUNNING, Scan: createScanWithAge(48 * time.Hour)}.Build(),
				storage.VirtualMachine_builder{Id: "vm3", ClusterId: "cluster2", Name: "vm-3", State: storage.VirtualMachine_STOPPED, Scan: createScanWithAge(1 * time.Hour)}.Build(),
			},
			expectedClustersWithRunningVMs: 1, // cluster1 (both VMs running)
			expectedTotalVMs:               3,
			expectedVMsWithActiveAgents:    2, // vm1 and vm3 have recent scans (vm2 scan is old)
		},
		"should count multiple distinct clusters with running VMs": {
			vms: []*storage.VirtualMachine{
				storage.VirtualMachine_builder{Id: "vm1", ClusterId: "cluster1", Name: "vm-1", State: storage.VirtualMachine_RUNNING, Scan: createScanWithAge(1 * time.Hour)}.Build(),
				storage.VirtualMachine_builder{Id: "vm2", ClusterId: "cluster2", Name: "vm-2", State: storage.VirtualMachine_RUNNING, Scan: createScanWithAge(1 * time.Hour)}.Build(),
				storage.VirtualMachine_builder{Id: "vm3", ClusterId: "cluster3", Name: "vm-3", State: storage.VirtualMachine_RUNNING, Scan: nil}.Build(),
			},
			expectedClustersWithRunningVMs: 3,
			expectedTotalVMs:               3,
			expectedVMsWithActiveAgents:    2, // vm1 and vm2 have recent scans
		},
		"should exclude VMs with empty cluster id from cluster count only": {
			vms: []*storage.VirtualMachine{
				storage.VirtualMachine_builder{Id: "vm1", ClusterId: "cluster1", Name: "vm-1", State: storage.VirtualMachine_RUNNING, Scan: createScanWithAge(1 * time.Hour)}.Build(),
				storage.VirtualMachine_builder{Id: "vm2", ClusterId: "", Name: "vm-orphan", State: storage.VirtualMachine_RUNNING, Scan: createScanWithAge(1 * time.Hour)}.Build(),
				storage.VirtualMachine_builder{Id: "vm3", ClusterId: "cluster2", Name: "vm-2", State: storage.VirtualMachine_RUNNING}.Build(),
			},
			expectedClustersWithRunningVMs: 2, // Empty cluster_id excluded
			expectedTotalVMs:               3, // All VMs counted
			expectedVMsWithActiveAgents:    2, // vm1 and vm2 have recent scans
		},
		"should handle complex mixed scenario with various scan ages": {
			vms: []*storage.VirtualMachine{
				storage.VirtualMachine_builder{Id: "vm1", ClusterId: "cluster1", Name: "vm-1", State: storage.VirtualMachine_RUNNING, Scan: createScanWithAge(1 * time.Hour)}.Build(),
				storage.VirtualMachine_builder{Id: "vm2", ClusterId: "cluster1", Name: "vm-2", State: storage.VirtualMachine_STOPPED, Scan: createScanWithAge(48 * time.Hour)}.Build(),
				storage.VirtualMachine_builder{Id: "vm3", ClusterId: "", Name: "vm-orphan", State: storage.VirtualMachine_RUNNING, Scan: nil}.Build(),
				storage.VirtualMachine_builder{Id: "vm4", ClusterId: "cluster2", Name: "vm-4", State: storage.VirtualMachine_UNKNOWN, Scan: createScanWithAge(1 * time.Hour)}.Build(),
				storage.VirtualMachine_builder{Id: "vm5", ClusterId: "cluster3", Name: "vm-5", State: storage.VirtualMachine_RUNNING, Scan: nil}.Build(),
			},
			expectedClustersWithRunningVMs: 2, // cluster1 and cluster3
			expectedTotalVMs:               5, // All VMs
			expectedVMsWithActiveAgents:    2, // vm1 and vm4 have recent scans (vm2 scan is old)
		},
		"should handle VMs with nil and invalid scan timestamps": {
			vms: []*storage.VirtualMachine{
				storage.VirtualMachine_builder{Id: "vm1", ClusterId: "cluster1", Name: "vm-1", State: storage.VirtualMachine_RUNNING, Scan: createScanWithAge(1 * time.Hour)}.Build(),
				storage.VirtualMachine_builder{Id: "vm2", ClusterId: "cluster1", Name: "vm-2", State: storage.VirtualMachine_RUNNING, Scan: nil}.Build(),
				storage.VirtualMachine_builder{Id: "vm3", ClusterId: "cluster2", Name: "vm-3", State: storage.VirtualMachine_RUNNING, Scan: storage.VirtualMachineScan_builder{ScanTime: nil}.Build()}.Build(),
			},
			expectedClustersWithRunningVMs: 2, // cluster1 and cluster2
			expectedTotalVMs:               3,
			expectedVMsWithActiveAgents:    1, // Only vm1 has valid recent scan
		},
		"should include VM with scan exactly at 24h boundary": {
			vms: []*storage.VirtualMachine{
				storage.VirtualMachine_builder{Id: "vm1", ClusterId: "cluster1", Name: "vm-1", State: storage.VirtualMachine_RUNNING, Scan: createScanWithAge(24 * time.Hour)}.Build(),
				storage.VirtualMachine_builder{Id: "vm2", ClusterId: "cluster1", Name: "vm-2", State: storage.VirtualMachine_RUNNING, Scan: createScanWithAge(1 * time.Hour)}.Build(),
			},
			expectedClustersWithRunningVMs: 1,
			expectedTotalVMs:               2,
			expectedVMsWithActiveAgents:    2, // Both within threshold (boundary is inclusive: <=24h)
		},
		"should include VM with scan just under 24h threshold": {
			vms: []*storage.VirtualMachine{
				storage.VirtualMachine_builder{Id: "vm1", ClusterId: "cluster1", Name: "vm-1", State: storage.VirtualMachine_RUNNING, Scan: createScanWithAge(23 * time.Hour)}.Build(),
				storage.VirtualMachine_builder{Id: "vm2", ClusterId: "cluster2", Name: "vm-2", State: storage.VirtualMachine_RUNNING, Scan: createScanWithAge(25 * time.Hour)}.Build(),
			},
			expectedClustersWithRunningVMs: 2,
			expectedTotalVMs:               2,
			expectedVMsWithActiveAgents:    1, // Only vm1 (23h ago) is within threshold
		},
		"should exclude VM with scan just over 24h threshold": {
			vms: []*storage.VirtualMachine{
				storage.VirtualMachine_builder{Id: "vm1", ClusterId: "cluster1", Name: "vm-1", State: storage.VirtualMachine_RUNNING, Scan: createScanWithAge(25 * time.Hour)}.Build(),
				storage.VirtualMachine_builder{Id: "vm2", ClusterId: "cluster1", Name: "vm-2", State: storage.VirtualMachine_RUNNING, Scan: createScanWithAge(1 * time.Hour)}.Build(),
			},
			expectedClustersWithRunningVMs: 1,
			expectedTotalVMs:               2,
			expectedVMsWithActiveAgents:    1, // Only vm2 (recent) counts, vm1 (25h ago) excluded
		},
		"should return zero active agents when all VMs have old scans": {
			vms: []*storage.VirtualMachine{
				storage.VirtualMachine_builder{Id: "vm1", ClusterId: "cluster1", Name: "vm-1", State: storage.VirtualMachine_RUNNING, Scan: createScanWithAge(48 * time.Hour)}.Build(),
				storage.VirtualMachine_builder{Id: "vm2", ClusterId: "cluster2", Name: "vm-2", State: storage.VirtualMachine_RUNNING, Scan: createScanWithAge(25 * time.Hour)}.Build(),
				storage.VirtualMachine_builder{Id: "vm3", ClusterId: "cluster3", Name: "vm-3", State: storage.VirtualMachine_RUNNING, Scan: createScanWithAge(48 * time.Hour)}.Build(),
			},
			expectedClustersWithRunningVMs: 3, // All running VMs count for cluster metric
			expectedTotalVMs:               3, // All VMs count for total
			expectedVMsWithActiveAgents:    0, // None have recent scans
		},
		"should handle mixed scan ages across boundary": {
			vms: []*storage.VirtualMachine{
				storage.VirtualMachine_builder{Id: "vm1", ClusterId: "cluster1", Name: "vm-recent", State: storage.VirtualMachine_RUNNING, Scan: createScanWithAge(1 * time.Hour)}.Build(),     // 1h ago - counted
				storage.VirtualMachine_builder{Id: "vm2", ClusterId: "cluster1", Name: "vm-justunder", State: storage.VirtualMachine_RUNNING, Scan: createScanWithAge(23 * time.Hour)}.Build(), // 23h ago - counted
				storage.VirtualMachine_builder{Id: "vm3", ClusterId: "cluster1", Name: "vm-boundary", State: storage.VirtualMachine_RUNNING, Scan: createScanWithAge(24 * time.Hour)}.Build(),  // 24h ago - counted
				storage.VirtualMachine_builder{Id: "vm4", ClusterId: "cluster1", Name: "vm-justover", State: storage.VirtualMachine_RUNNING, Scan: createScanWithAge(25 * time.Hour)}.Build(),  // 25h ago - excluded
				storage.VirtualMachine_builder{Id: "vm5", ClusterId: "cluster1", Name: "vm-old", State: storage.VirtualMachine_RUNNING, Scan: createScanWithAge(48 * time.Hour)}.Build(),       // 48h ago - excluded
			},
			expectedClustersWithRunningVMs: 1, // All in cluster1
			expectedTotalVMs:               5, // All VMs counted
			expectedVMsWithActiveAgents:    3, // vm1, vm2, vm3 within threshold
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := sac.WithAllAccess(context.Background())

			ds := &mockDataStore{
				vms: tc.vms,
			}

			gatherer := gatherWithTime(ds, arbitraryNowFunc)
			props, err := gatherer(ctx)

			require.NoError(t, err)
			require.NotNil(t, props)

			assert.Equal(t, tc.expectedClustersWithRunningVMs, props[metricClustersWithVMs],
				"Mismatch in clusters with running VMs count")
			assert.Equal(t, tc.expectedTotalVMs, props[metricTotalVMs],
				"Mismatch in total VMs count")
			assert.Equal(t, tc.expectedVMsWithActiveAgents, props[metricVMsWithActiveAgents],
				"Mismatch in VMs with active agents count")
		})
	}
}

func TestVirtualMachineTelemetryWithFeatureFlagDisabled(t *testing.T) {
	// Disable the feature flag using t.Setenv for automatic cleanup
	t.Setenv(features.VirtualMachines.EnvVar(), "false")

	ctx := sac.WithAllAccess(context.Background())

	// Create a mock datastore with VMs (which should NOT be queried)
	vm := &storage.VirtualMachine{}
	vm.SetId("vm1")
	vm.SetClusterId("cluster1")
	vm.SetName("test-vm")
	vm.SetState(storage.VirtualMachine_RUNNING)
	vm.SetScan(createScanWithAge(1 * time.Hour))
	vm2 := &storage.VirtualMachine{}
	vm2.SetId("vm2")
	vm2.SetClusterId("cluster2")
	vm2.SetName("test-vm2")
	vm2.SetState(storage.VirtualMachine_RUNNING)
	vm2.SetScan(createScanWithAge(1 * time.Hour))
	ds := &mockDataStore{
		vms: []*storage.VirtualMachine{
			vm,
			vm2,
		},
	}

	gatherer := gatherWithTime(ds, arbitraryNowFunc)
	props, err := gatherer(ctx)

	require.NoError(t, err)
	require.NotNil(t, props)

	// When feature flag is disabled, should return empty map
	// No database query should have been performed
	assert.Empty(t, props, "Should return empty map when feature flag is disabled")
}

// createScanWithAge creates a scan with a timestamp at the specified duration before arbitraryNow.
// For example, createScanWithAge(1 * time.Hour) creates a scan from 1 hour ago.
func createScanWithAge(age time.Duration) *storage.VirtualMachineScan {
	scanTime := arbitraryNow.Add(-age)
	vms := &storage.VirtualMachineScan{}
	vms.SetScanTime(protocompat.ConvertTimeToTimestampOrNil(&scanTime))
	return vms
}

type mockDataStore struct {
	vms []*storage.VirtualMachine
	err error
}

func (m *mockDataStore) CountVirtualMachines(ctx context.Context, query *v1.Query) (int, error) {
	return 0, nil
}

func (m *mockDataStore) GetVirtualMachine(ctx context.Context, id string) (*storage.VirtualMachine, bool, error) {
	return nil, false, nil
}

func (m *mockDataStore) UpsertVirtualMachine(ctx context.Context, virtualMachine *storage.VirtualMachine) error {
	return nil
}

func (m *mockDataStore) UpdateVirtualMachineScan(ctx context.Context, vmID string, scan *storage.VirtualMachineScan) error {
	return nil
}

func (m *mockDataStore) DeleteVirtualMachines(ctx context.Context, ids ...string) error {
	return nil
}

func (m *mockDataStore) Exists(ctx context.Context, id string) (bool, error) {
	return false, nil
}

func (m *mockDataStore) SearchRawVirtualMachines(ctx context.Context, query *v1.Query) ([]*storage.VirtualMachine, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.vms, nil
}

func (m *mockDataStore) Walk(ctx context.Context, fn func(vm *storage.VirtualMachine) error) error {
	if m.err != nil {
		return m.err
	}
	for _, vm := range m.vms {
		if err := fn(vm); err != nil {
			return err
		}
	}
	return nil
}
