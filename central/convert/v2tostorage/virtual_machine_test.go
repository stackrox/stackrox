package v2tostorage

import (
	"testing"
	"time"

	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestVirtualMachine(t *testing.T) {
	timestamp := timestamppb.New(time.Now())

	tests := []struct {
		name     string
		input    *v2.VirtualMachine
		expected *storage.VirtualMachine
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "complete virtual machine",
			input: &v2.VirtualMachine{
				Id:          "vm-123",
				Namespace:   "default",
				Name:        "test-vm",
				ClusterId:   "cluster-456",
				ClusterName: "test-cluster",
				VsockCid:    int32(81),
				State:       v2.VirtualMachine_RUNNING,
				Scan: &v2.VirtualMachineScan{
					ScannerVersion: "1.0.0",
					ScanTime:       timestamp,
					Components: []*v2.ScanComponent{
						{
							Name:    "test-component",
							Version: "1.0.0",
							Vulns: []*v2.EmbeddedVulnerability{
								{
									Cve:     "CVE-2023-1234",
									Summary: "Test vulnerability",
								},
							},
						},
					},
				},
				LastUpdated: timestamp,
			},
			expected: &storage.VirtualMachine{
				Id:          "vm-123",
				Namespace:   "default",
				Name:        "test-vm",
				ClusterId:   "cluster-456",
				ClusterName: "test-cluster",
				VsockCid:    int32(81),
				State:       storage.VirtualMachine_RUNNING,
				Scan: &storage.VirtualMachineScan{
					ScannerVersion: "1.0.0",
					ScanTime:       timestamp,
					Components: []*storage.EmbeddedVirtualMachineScanComponent{
						{
							Name:    "test-component",
							Version: "1.0.0",
							Vulns: []*storage.EmbeddedVirtualMachineVulnerability{
								{
									Cve: "CVE-2023-1234",
								},
							},
						},
					},
				},
				LastUpdated: timestamp,
			},
		},
		{
			name: "stopped virtual machine",
			input: &v2.VirtualMachine{
				Id:        "vm-stopped",
				Namespace: "test",
				Name:      "stopped-vm",
				State:     v2.VirtualMachine_STOPPED,
			},
			expected: &storage.VirtualMachine{
				Id:        "vm-stopped",
				Namespace: "test",
				Name:      "stopped-vm",
				State:     storage.VirtualMachine_STOPPED,
			},
		},
		{
			name: "minimal virtual machine",
			input: &v2.VirtualMachine{
				Id:        "vm-minimal",
				Namespace: "test",
				Name:      "minimal-vm",
			},
			expected: &storage.VirtualMachine{
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

func TestVirtualMachineScan(t *testing.T) {
	timestamp := timestamppb.New(time.Now())

	tests := []struct {
		name     string
		input    *v2.VirtualMachineScan
		expected *storage.VirtualMachineScan
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "complete scan",
			input: &v2.VirtualMachineScan{
				ScannerVersion: "2.0.0",
				ScanTime:       timestamp,
				Components: []*v2.ScanComponent{
					{
						Name:    "component1",
						Version: "1.0.0",
					},
				},
			},
			expected: &storage.VirtualMachineScan{
				ScannerVersion: "2.0.0",
				ScanTime:       timestamp,
				Components: []*storage.EmbeddedVirtualMachineScanComponent{
					{
						Name:    "component1",
						Version: "1.0.0",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := VirtualMachineScan(tt.input)
			protoassert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertVirtualMachineState(t *testing.T) {
	tests := []struct {
		name     string
		input    v2.VirtualMachine_State
		expected storage.VirtualMachine_State
	}{
		{
			name:     "UNKNOWN",
			input:    v2.VirtualMachine_UNKNOWN,
			expected: storage.VirtualMachine_UNKNOWN,
		},
		{
			name:     "STOPPED",
			input:    v2.VirtualMachine_STOPPED,
			expected: storage.VirtualMachine_STOPPED,
		},
		{
			name:     "RUNNING",
			input:    v2.VirtualMachine_RUNNING,
			expected: storage.VirtualMachine_RUNNING,
		},
		{
			name:     "Other",
			input:    v2.VirtualMachine_State(-1),
			expected: storage.VirtualMachine_UNKNOWN,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertVirtualMachineState(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVirtualMachineScanComponents(t *testing.T) {
	tests := []struct {
		name     string
		input    []*v2.ScanComponent
		expected []*storage.EmbeddedVirtualMachineScanComponent
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "minimal component",
			input: []*v2.ScanComponent{
				{
					Name:    "component1",
					Version: "1.0.0",
				},
			},
			expected: []*storage.EmbeddedVirtualMachineScanComponent{
				{
					Name:    "component1",
					Version: "1.0.0",
				},
			},
		},
		{
			name: "nil and non-nil component",
			input: []*v2.ScanComponent{
				nil,
				{
					Name:    "component1",
					Version: "1.0.0",
				},
			},
			expected: []*storage.EmbeddedVirtualMachineScanComponent{
				{
					Name:    "component1",
					Version: "1.0.0",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := VirtualMachineScanComponents(tt.input)
			protoassert.SlicesEqual(t, tt.expected, result)
		})
	}
}

func TestVirtualMachineScanComponent(t *testing.T) {
	tests := []struct {
		name     string
		input    *v2.ScanComponent
		expected *storage.EmbeddedVirtualMachineScanComponent
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "minimal component",
			input: &v2.ScanComponent{
				Name:    "component1",
				Version: "1.0.0",
			},
			expected: &storage.EmbeddedVirtualMachineScanComponent{
				Name:    "component1",
				Version: "1.0.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := VirtualMachineScanComponent(tt.input)
			protoassert.Equal(t, tt.expected, result)
		})
	}
}

func TestEmbeddedVirtualMachineVulnerabilities(t *testing.T) {
	tests := []struct {
		name     string
		input    []*v2.EmbeddedVulnerability
		expected []*storage.EmbeddedVirtualMachineVulnerability
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "vulnerability with fixed by",
			input: []*v2.EmbeddedVulnerability{
				nil,
				{
					Cve: "CVE-2023-1234",
				},
			},
			expected: []*storage.EmbeddedVirtualMachineVulnerability{
				{
					Cve: "CVE-2023-1234",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EmbeddedVirtualMachineVulnerabilities(tt.input)
			protoassert.SlicesEqual(t, tt.expected, result)
		})
	}
}

func TestEmbeddedVirtualMachineVulnerability(t *testing.T) {
	tests := []struct {
		name     string
		input    *v2.EmbeddedVulnerability
		expected *storage.EmbeddedVirtualMachineVulnerability
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "vulnerability with fixed by",
			input: &v2.EmbeddedVulnerability{
				Cve: "CVE-2023-1234",
			},
			expected: &storage.EmbeddedVirtualMachineVulnerability{
				Cve: "CVE-2023-1234",
			},
		},
		{
			name: "vulnerability without fixed by",
			input: &v2.EmbeddedVulnerability{
				Cve: "CVE-2023-5678",
			},
			expected: &storage.EmbeddedVirtualMachineVulnerability{
				Cve: "CVE-2023-5678",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EmbeddedVirtualMachineVulnerability(tt.input)
			protoassert.Equal(t, tt.expected, result)
		})
	}
}
