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

// Helper function to create scan with recent timestamp (within last 24h)
func createRecentScan() *storage.VirtualMachineScan {
	return &storage.VirtualMachineScan{
		ScanTime: protocompat.TimestampNow(), // Within last 24h
	}
}

// Helper function to create scan with old timestamp (older than 24h)
func createOldScan() *storage.VirtualMachineScan {
	twoDaysAgo := time.Now().Add(-48 * time.Hour)
	return &storage.VirtualMachineScan{
		ScanTime: protocompat.ConvertTimeToTimestampOrNil(&twoDaysAgo), // Older than 24h
	}
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
				{Id: "vm1", ClusterId: "cluster1", Name: "test-vm", State: storage.VirtualMachine_RUNNING, Scan: createRecentScan()},
			},
			expectedClustersWithRunningVMs: 1,
			expectedTotalVMs:               1,
			expectedVMsWithActiveAgents:    1,
		},
		"should count total VMs correctly including stopped ones": {
			vms: []*storage.VirtualMachine{
				{Id: "vm1", ClusterId: "cluster1", Name: "vm-1", State: storage.VirtualMachine_RUNNING},
				{Id: "vm2", ClusterId: "cluster1", Name: "vm-2", State: storage.VirtualMachine_STOPPED},
				{Id: "vm3", ClusterId: "cluster2", Name: "vm-3", State: storage.VirtualMachine_UNKNOWN},
			},
			expectedClustersWithRunningVMs: 1, // Only cluster1 has RUNNING VM
			expectedTotalVMs:               3, // All VMs counted
			expectedVMsWithActiveAgents:    0, // None have recent scan data
		},
		"should exclude VMs with old scans from active agent count": {
			vms: []*storage.VirtualMachine{
				{Id: "vm1", ClusterId: "cluster1", Name: "vm-1", State: storage.VirtualMachine_RUNNING, Scan: createRecentScan()},
				{Id: "vm2", ClusterId: "cluster1", Name: "vm-2", State: storage.VirtualMachine_RUNNING, Scan: createOldScan()},
				{Id: "vm3", ClusterId: "cluster2", Name: "vm-3", State: storage.VirtualMachine_STOPPED, Scan: createRecentScan()},
			},
			expectedClustersWithRunningVMs: 1, // cluster1 (both VMs running)
			expectedTotalVMs:               3,
			expectedVMsWithActiveAgents:    2, // vm1 and vm3 have recent scans (vm2 scan is old)
		},
		"should count multiple distinct clusters with running VMs": {
			vms: []*storage.VirtualMachine{
				{Id: "vm1", ClusterId: "cluster1", Name: "vm-1", State: storage.VirtualMachine_RUNNING, Scan: createRecentScan()},
				{Id: "vm2", ClusterId: "cluster2", Name: "vm-2", State: storage.VirtualMachine_RUNNING, Scan: createRecentScan()},
				{Id: "vm3", ClusterId: "cluster3", Name: "vm-3", State: storage.VirtualMachine_RUNNING, Scan: nil},
			},
			expectedClustersWithRunningVMs: 3,
			expectedTotalVMs:               3,
			expectedVMsWithActiveAgents:    2, // vm1 and vm2 have recent scans
		},
		"should exclude VMs with empty cluster id from cluster count only": {
			vms: []*storage.VirtualMachine{
				{Id: "vm1", ClusterId: "cluster1", Name: "vm-1", State: storage.VirtualMachine_RUNNING, Scan: createRecentScan()},
				{Id: "vm2", ClusterId: "", Name: "vm-orphan", State: storage.VirtualMachine_RUNNING, Scan: createRecentScan()},
				{Id: "vm3", ClusterId: "cluster2", Name: "vm-2", State: storage.VirtualMachine_RUNNING},
			},
			expectedClustersWithRunningVMs: 2, // Empty cluster_id excluded
			expectedTotalVMs:               3, // All VMs counted
			expectedVMsWithActiveAgents:    2, // vm1 and vm2 have recent scans
		},
		"should handle complex mixed scenario with various scan ages": {
			vms: []*storage.VirtualMachine{
				{Id: "vm1", ClusterId: "cluster1", Name: "vm-1", State: storage.VirtualMachine_RUNNING, Scan: createRecentScan()},
				{Id: "vm2", ClusterId: "cluster1", Name: "vm-2", State: storage.VirtualMachine_STOPPED, Scan: createOldScan()},
				{Id: "vm3", ClusterId: "", Name: "vm-orphan", State: storage.VirtualMachine_RUNNING, Scan: nil},
				{Id: "vm4", ClusterId: "cluster2", Name: "vm-4", State: storage.VirtualMachine_UNKNOWN, Scan: createRecentScan()},
				{Id: "vm5", ClusterId: "cluster3", Name: "vm-5", State: storage.VirtualMachine_RUNNING, Scan: nil},
			},
			expectedClustersWithRunningVMs: 2, // cluster1 and cluster3
			expectedTotalVMs:               5, // All VMs
			expectedVMsWithActiveAgents:    2, // vm1 and vm4 have recent scans (vm2 scan is old)
		},
		"should handle VMs with nil and invalid scan timestamps": {
			vms: []*storage.VirtualMachine{
				{Id: "vm1", ClusterId: "cluster1", Name: "vm-1", State: storage.VirtualMachine_RUNNING, Scan: createRecentScan()},
				{Id: "vm2", ClusterId: "cluster1", Name: "vm-2", State: storage.VirtualMachine_RUNNING, Scan: nil},
				{Id: "vm3", ClusterId: "cluster2", Name: "vm-3", State: storage.VirtualMachine_RUNNING, Scan: &storage.VirtualMachineScan{ScanTime: nil}},
			},
			expectedClustersWithRunningVMs: 2, // cluster1 and cluster2
			expectedTotalVMs:               3,
			expectedVMsWithActiveAgents:    1, // Only vm1 has valid recent scan
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := sac.WithAllAccess(context.Background())

			ds := &mockDataStore{
				vms: tc.vms,
			}

			gatherer := Gather(ds)
			props, err := gatherer(ctx)

			require.NoError(t, err)
			require.NotNil(t, props)

			assert.Equal(t, tc.expectedClustersWithRunningVMs, props["Total Secured Clusters With Virtual Machines"],
				"Mismatch in clusters with running VMs count")
			assert.Equal(t, tc.expectedTotalVMs, props["Total Virtual Machines"],
				"Mismatch in total VMs count")
			assert.Equal(t, tc.expectedVMsWithActiveAgents, props["Total Virtual Machines With Active Agents (Last 24h)"],
				"Mismatch in VMs with active agents count")
		})
	}
}

func TestVirtualMachineTelemetryWithFeatureFlagDisabled(t *testing.T) {
	// Disable the feature flag using t.Setenv for automatic cleanup
	t.Setenv(features.VirtualMachines.EnvVar(), "false")

	ctx := sac.WithAllAccess(context.Background())

	// Create a mock datastore with VMs (which should NOT be queried)
	ds := &mockDataStore{
		vms: []*storage.VirtualMachine{
			{Id: "vm1", ClusterId: "cluster1", Name: "test-vm", State: storage.VirtualMachine_RUNNING, Scan: createRecentScan()},
			{Id: "vm2", ClusterId: "cluster2", Name: "test-vm2", State: storage.VirtualMachine_RUNNING, Scan: createRecentScan()},
		},
	}

	gatherer := Gather(ds)
	props, err := gatherer(ctx)

	require.NoError(t, err)
	require.NotNil(t, props)

	// When feature flag is disabled, should return empty map
	// No database query should have been performed
	assert.Empty(t, props, "Should return empty map when feature flag is disabled")
}
