package storagetov2

import (
	"testing"
	"time"

	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	testScanOS = "rhel/9"
)

var (
	testScanTime = time.Date(2025, time.September, 15, 16, 17, 18, 912345678, time.UTC)
)

func TestVirtualMachine(t *testing.T) {
	timestamp := timestamppb.New(time.Now())

	tests := []struct {
		name     string
		input    *storage.VirtualMachine
		expected *v2.VirtualMachine
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "complete virtual machine",
			input: &storage.VirtualMachine{
				Id:          "vm-123",
				Namespace:   "default",
				Name:        "test-vm",
				ClusterId:   "cluster-456",
				ClusterName: "test-cluster",
				VsockCid:    int32(42),
				State:       storage.VirtualMachine_RUNNING,
				LastUpdated: timestamp,
				Scan:        &storage.VirtualMachineScan{},
			},
			expected: &v2.VirtualMachine{
				Id:          "vm-123",
				Namespace:   "default",
				Name:        "test-vm",
				ClusterId:   "cluster-456",
				ClusterName: "test-cluster",
				VsockCid:    int32(42),
				State:       v2.VirtualMachine_RUNNING,
				LastUpdated: timestamp,
				Scan:        &v2.VirtualMachineScan{},
			},
		},
		{
			name: "stopped virtual machine",
			input: &storage.VirtualMachine{
				Id:        "vm-stopped",
				Namespace: "test",
				Name:      "stopped-vm",
				State:     storage.VirtualMachine_STOPPED,
			},
			expected: &v2.VirtualMachine{
				Id:        "vm-stopped",
				Namespace: "test",
				Name:      "stopped-vm",
				State:     v2.VirtualMachine_STOPPED,
			},
		},
		{
			name: "minimal virtual machine",
			input: &storage.VirtualMachine{
				Id:        "vm-minimal",
				Namespace: "test",
				Name:      "minimal-vm",
			},
			expected: &v2.VirtualMachine{
				Id:        "vm-minimal",
				Namespace: "test",
				Name:      "minimal-vm",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := VirtualMachine(tt.input)
			protoassert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertVirtualMachineState(t *testing.T) {
	tests := []struct {
		name     string
		input    storage.VirtualMachine_State
		expected v2.VirtualMachine_State
	}{
		{
			name:     "UNKNOWN",
			input:    storage.VirtualMachine_UNKNOWN,
			expected: v2.VirtualMachine_UNKNOWN,
		},
		{
			name:     "STOPPED",
			input:    storage.VirtualMachine_STOPPED,
			expected: v2.VirtualMachine_STOPPED,
		},
		{
			name:     "RUNNING",
			input:    storage.VirtualMachine_RUNNING,
			expected: v2.VirtualMachine_RUNNING,
		},
		{
			name:     "Other",
			input:    storage.VirtualMachine_State(-1),
			expected: v2.VirtualMachine_UNKNOWN,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertVirtualMachineState(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVirtualMachineScan(t *testing.T) {
	tests := []struct {
		name     string
		input    *storage.VirtualMachineScan
		expected *v2.VirtualMachineScan
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "scan without component",
			input: &storage.VirtualMachineScan{
				ScanTime:        protocompat.ConvertTimeToTimestampOrNil(&testScanTime),
				OperatingSystem: testScanOS,
				Notes: []storage.VirtualMachineScan_Note{
					storage.VirtualMachineScan_UNSET,
				},
				Components: []*storage.EmbeddedVirtualMachineScanComponent{
					{
						Name:      testComponentName,
						Version:   testComponentVersion,
						RiskScore: testComponentRiskScore,
						Vulnerabilities: []*storage.VirtualMachineVulnerability{
							storageVirtualMachineTestVuln,
						},
					},
				},
			},
			expected: &v2.VirtualMachineScan{
				ScanTime:        protocompat.ConvertTimeToTimestampOrNil(&testScanTime),
				OperatingSystem: testScanOS,
				Notes: []v2.VirtualMachineScan_Note{
					v2.VirtualMachineScan_UNSET,
				},
				Components: []*v2.ScanComponent{
					{
						Name:      testComponentName,
						Version:   testComponentVersion,
						RiskScore: testComponentRiskScore,
						Vulns: []*v2.EmbeddedVulnerability{
							v2VirtualMachineTestVuln,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(it *testing.T) {
			result := VirtualMachineScan(tt.input)
			protoassert.Equal(it, tt.expected, result)
		})
	}
}

func TestVirtualMachineScanNotes(t *testing.T) {
	tests := []struct {
		name     string
		input    []storage.VirtualMachineScan_Note
		expected []v2.VirtualMachineScan_Note
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "value mix",
			input: []storage.VirtualMachineScan_Note{
				storage.VirtualMachineScan_UNSET,
				storage.VirtualMachineScan_Note(-1),
			},
			expected: []v2.VirtualMachineScan_Note{
				v2.VirtualMachineScan_UNSET,
				v2.VirtualMachineScan_UNSET,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(it *testing.T) {
			result := VirtualMachineScanNotes(tc.input)
			assert.Equal(it, tc.expected, result)
		})
	}
}

func TestConvertVirtualMachineScanNote(t *testing.T) {
	tests := []struct {
		name     string
		input    storage.VirtualMachineScan_Note
		expected v2.VirtualMachineScan_Note
	}{
		{
			name:     "UNSET",
			input:    storage.VirtualMachineScan_UNSET,
			expected: v2.VirtualMachineScan_UNSET,
		},
		{
			name:     "default",
			input:    storage.VirtualMachineScan_Note(-1),
			expected: v2.VirtualMachineScan_UNSET,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(it *testing.T) {
			result := convertVirtualMachineScanNote(tc.input)
			assert.Equal(it, tc.expected, result)
		})
	}
}
