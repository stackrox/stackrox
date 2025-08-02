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
				Scan: &v2.VirtualMachineScan{
					ScannerVersion: "1.0.0",
					ScanTime:       timestamp,
					Components: []*v2.ScanComponent{
						{
							Name:    "test-component",
							Version: "1.0.0",
							Vulns: []*v2.EmbeddedVulnerability{
								{
									Cve:      "CVE-2023-1234",
									Summary:  "Test vulnerability",
									Severity: v2.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
								},
							},
						},
					},
					DataSource: &v2.DataSource{
						Id:     "ds-1",
						Name:   "test-datasource",
						Mirror: "mirror.example.com",
					},
					Notes: []v2.VirtualMachineScan_Note{
						v2.VirtualMachineScan_OS_UNAVAILABLE,
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
				Scan: &storage.VirtualMachineScan{
					ScannerVersion: "1.0.0",
					ScanTime:       timestamp,
					Components: []*storage.EmbeddedImageScanComponent{
						{
							Name:    "test-component",
							Version: "1.0.0",
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve:      "CVE-2023-1234",
									Summary:  "Test vulnerability",
									Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
								},
							},
						},
					},
					DataSource: &storage.DataSource{
						Id:     "ds-1",
						Name:   "test-datasource",
						Mirror: "mirror.example.com",
					},
					Notes: []storage.VirtualMachineScan_Note{
						storage.VirtualMachineScan_OS_UNAVAILABLE,
					},
				},
				LastUpdated: timestamp,
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
						Name:     "component1",
						Version:  "1.0.0",
						Location: "/usr/bin/component1",
						Source:   v2.SourceType_OS,
					},
				},
				DataSource: &v2.DataSource{
					Id:     "ds-test",
					Name:   "test-source",
					Mirror: "mirror.test.com",
				},
				Notes: []v2.VirtualMachineScan_Note{
					v2.VirtualMachineScan_PARTIAL_SCAN_DATA,
					v2.VirtualMachineScan_OS_CVES_STALE,
				},
			},
			expected: &storage.VirtualMachineScan{
				ScannerVersion: "2.0.0",
				ScanTime:       timestamp,
				Components: []*storage.EmbeddedImageScanComponent{
					{
						Name:     "component1",
						Version:  "1.0.0",
						Location: "/usr/bin/component1",
						Source:   storage.SourceType_OS,
					},
				},
				DataSource: &storage.DataSource{
					Id:     "ds-test",
					Name:   "test-source",
					Mirror: "mirror.test.com",
				},
				Notes: []storage.VirtualMachineScan_Note{
					storage.VirtualMachineScan_PARTIAL_SCAN_DATA,
					storage.VirtualMachineScan_OS_CVES_STALE,
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

func TestConvertVirtualMachineScanNotes(t *testing.T) {
	tests := []struct {
		name     string
		input    []v2.VirtualMachineScan_Note
		expected []storage.VirtualMachineScan_Note
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty input",
			input:    []v2.VirtualMachineScan_Note{},
			expected: nil,
		},
		{
			name: "multiple notes",
			input: []v2.VirtualMachineScan_Note{
				v2.VirtualMachineScan_OS_UNAVAILABLE,
				v2.VirtualMachineScan_PARTIAL_SCAN_DATA,
				v2.VirtualMachineScan_OS_CVES_STALE,
			},
			expected: []storage.VirtualMachineScan_Note{
				storage.VirtualMachineScan_OS_UNAVAILABLE,
				storage.VirtualMachineScan_PARTIAL_SCAN_DATA,
				storage.VirtualMachineScan_OS_CVES_STALE,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertVirtualMachineScanNotes(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
