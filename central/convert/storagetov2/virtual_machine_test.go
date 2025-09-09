package storagetov2

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
				Scan: &storage.VirtualMachineScan{
					ScannerVersion: "1.0.0",
					ScanTime:       timestamp,
					Components: []*storage.EmbeddedVirtualMachineScanComponent{
						{
							Name:    "test-component",
							Version: "1.0.0",
							Vulns: []*storage.EmbeddedVirtualMachineVulnerability{
								{
									Cve:      "CVE-2023-1234",
									Summary:  "Test vulnerability",
									Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
								},
							},
						},
					},
					DataSource: &storage.VirtualMachineScan_DataSource{
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
			expected: &v2.VirtualMachine{
				Id:          "vm-123",
				Namespace:   "default",
				Name:        "test-vm",
				ClusterId:   "cluster-456",
				ClusterName: "test-cluster",
				VsockCid:    int32(42),
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

func TestVirtualMachineScan(t *testing.T) {
	timestamp := timestamppb.New(time.Now())

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
			name: "complete scan",
			input: &storage.VirtualMachineScan{
				ScannerVersion: "2.0.0",
				ScanTime:       timestamp,
				Components: []*storage.EmbeddedVirtualMachineScanComponent{
					{
						Name:     "component1",
						Version:  "1.0.0",
						Location: "/usr/bin/component1",
						Source:   storage.EmbeddedVirtualMachineScanComponent_OS,
					},
				},
				DataSource: &storage.VirtualMachineScan_DataSource{
					Id:     "ds-test",
					Name:   "test-source",
					Mirror: "mirror.test.com",
				},
				Notes: []storage.VirtualMachineScan_Note{
					storage.VirtualMachineScan_PARTIAL_SCAN_DATA,
					storage.VirtualMachineScan_OS_CVES_STALE,
				},
			},
			expected: &v2.VirtualMachineScan{
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
		input    []storage.VirtualMachineScan_Note
		expected []v2.VirtualMachineScan_Note
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty input",
			input:    []storage.VirtualMachineScan_Note{},
			expected: nil,
		},
		{
			name: "multiple notes",
			input: []storage.VirtualMachineScan_Note{
				storage.VirtualMachineScan_OS_UNAVAILABLE,
				storage.VirtualMachineScan_PARTIAL_SCAN_DATA,
				storage.VirtualMachineScan_OS_CVES_STALE,
			},
			expected: []v2.VirtualMachineScan_Note{
				v2.VirtualMachineScan_OS_UNAVAILABLE,
				v2.VirtualMachineScan_PARTIAL_SCAN_DATA,
				v2.VirtualMachineScan_OS_CVES_STALE,
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

func TestVirtualMachineScanComponents(t *testing.T) {
	tests := []struct {
		name     string
		input    []*storage.EmbeddedVirtualMachineScanComponent
		expected []*v2.ScanComponent
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "minimal component",
			input: []*storage.EmbeddedVirtualMachineScanComponent{
				{
					Name:     "component1",
					Version:  "1.0.0",
					Location: "/usr/bin/component1",
					Source:   storage.EmbeddedVirtualMachineScanComponent_OS,
				},
			},
			expected: []*v2.ScanComponent{
				{
					Name:     "component1",
					Version:  "1.0.0",
					Location: "/usr/bin/component1",
					Source:   v2.SourceType_OS,
				},
			},
		},
		{
			name: "nil and non-nil component",
			input: []*storage.EmbeddedVirtualMachineScanComponent{
				nil,
				{
					Name:     "component1",
					Version:  "1.0.0",
					Location: "/usr/bin/component1",
					Source:   storage.EmbeddedVirtualMachineScanComponent_OS,
					SetTopCvss: &storage.EmbeddedVirtualMachineScanComponent_TopCvss{
						TopCvss: 5.5,
					},
				},
			},
			expected: []*v2.ScanComponent{
				{
					Name:     "component1",
					Version:  "1.0.0",
					Location: "/usr/bin/component1",
					Source:   v2.SourceType_OS,
					SetTopCvss: &v2.ScanComponent_TopCvss{
						TopCvss: 5.5,
					},
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
		input    *storage.EmbeddedVirtualMachineScanComponent
		expected *v2.ScanComponent
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "minimal component",
			input: &storage.EmbeddedVirtualMachineScanComponent{
				Name:     "component1",
				Version:  "1.0.0",
				Location: "/usr/bin/component1",
				Source:   storage.EmbeddedVirtualMachineScanComponent_OS,
			},
			expected: &v2.ScanComponent{
				Name:     "component1",
				Version:  "1.0.0",
				Location: "/usr/bin/component1",
				Source:   v2.SourceType_OS,
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

func TestVirtualMachineLicense(t *testing.T) {
	tests := []struct {
		name     string
		input    *storage.EmbeddedVirtualMachineScanComponent_License
		expected *v2.License
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "complete license",
			input: &storage.EmbeddedVirtualMachineScanComponent_License{
				Name: "Apache-2.0",
				Type: "permissive",
				Url:  "https://opensource.org/licenses/Apache-2.0",
			},
			expected: &v2.License{
				Name: "Apache-2.0",
				Type: "permissive",
				Url:  "https://opensource.org/licenses/Apache-2.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := VirtualMachineLicense(tt.input)
			protoassert.Equal(t, tt.expected, result)
		})
	}
}

func TestEmbeddedVirtualMachineVulnerabilities(t *testing.T) {
	publishedOn := timestamppb.New(time.Now().Add(-24 * time.Hour))
	lastModified := timestamppb.New(time.Now())

	tests := []struct {
		name     string
		input    []*storage.EmbeddedVirtualMachineVulnerability
		expected []*v2.EmbeddedVulnerability
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "vulnerability with fixed by",
			input: []*storage.EmbeddedVirtualMachineVulnerability{
				nil,
				{
					Cve:     "CVE-2023-1234",
					Summary: "Test vulnerability",
					Link:    "https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2023-1234",
					SetFixedBy: &storage.EmbeddedVirtualMachineVulnerability_FixedBy{
						FixedBy: "1.2.4",
					},
					VulnerabilityType: storage.EmbeddedVirtualMachineVulnerability_IMAGE_VULNERABILITY,
					Severity:          storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
					CvssV3: &storage.CVSSV3{
						Vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
						Score:  9.8,
					},
					PublishedOn:  publishedOn,
					LastModified: lastModified,
				},
			},
			expected: []*v2.EmbeddedVulnerability{
				{
					Cve:     "CVE-2023-1234",
					Summary: "Test vulnerability",
					Link:    "https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2023-1234",
					SetFixedBy: &v2.EmbeddedVulnerability_FixedBy{
						FixedBy: "1.2.4",
					},
					VulnerabilityType: v2.VulnerabilityType_IMAGE_VULNERABILITY,
					Severity:          v2.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
					CvssV3: &v2.CVSSV3{
						Vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
						Score:  9.8,
					},
					PublishedOn:  publishedOn,
					LastModified: lastModified,
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
	publishedOn := timestamppb.New(time.Now().Add(-24 * time.Hour))
	lastModified := timestamppb.New(time.Now())

	tests := []struct {
		name     string
		input    *storage.EmbeddedVirtualMachineVulnerability
		expected *v2.EmbeddedVulnerability
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "vulnerability with fixed by",
			input: &storage.EmbeddedVirtualMachineVulnerability{
				Cve:     "CVE-2023-1234",
				Summary: "Test vulnerability",
				Link:    "https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2023-1234",
				SetFixedBy: &storage.EmbeddedVirtualMachineVulnerability_FixedBy{
					FixedBy: "1.2.4",
				},
				VulnerabilityType: storage.EmbeddedVirtualMachineVulnerability_IMAGE_VULNERABILITY,
				Severity:          storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
				CvssV3: &storage.CVSSV3{
					Vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
					Score:  9.8,
				},
				PublishedOn:  publishedOn,
				LastModified: lastModified,
			},
			expected: &v2.EmbeddedVulnerability{
				Cve:     "CVE-2023-1234",
				Summary: "Test vulnerability",
				Link:    "https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2023-1234",
				SetFixedBy: &v2.EmbeddedVulnerability_FixedBy{
					FixedBy: "1.2.4",
				},
				VulnerabilityType: v2.VulnerabilityType_IMAGE_VULNERABILITY,
				Severity:          v2.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
				CvssV3: &v2.CVSSV3{
					Vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
					Score:  9.8,
				},
				PublishedOn:  publishedOn,
				LastModified: lastModified,
			},
		},
		{
			name: "vulnerability without fixed by",
			input: &storage.EmbeddedVirtualMachineVulnerability{
				Cve:               "CVE-2023-5678",
				Summary:           "Another test vulnerability",
				VulnerabilityType: storage.EmbeddedVirtualMachineVulnerability_NODE_VULNERABILITY,
				Severity:          storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
			},
			expected: &v2.EmbeddedVulnerability{
				Cve:               "CVE-2023-5678",
				Summary:           "Another test vulnerability",
				VulnerabilityType: v2.VulnerabilityType_NODE_VULNERABILITY,
				Severity:          v2.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
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

func TestConvertVirtualMachineSourceType(t *testing.T) {
	tests := []struct {
		name     string
		input    storage.EmbeddedVirtualMachineScanComponent_SourceType
		expected v2.SourceType
	}{
		{
			name:     "OS",
			input:    storage.EmbeddedVirtualMachineScanComponent_OS,
			expected: v2.SourceType_OS,
		},
		{
			name:     "PYTHON",
			input:    storage.EmbeddedVirtualMachineScanComponent_PYTHON,
			expected: v2.SourceType_PYTHON,
		},
		{
			name:     "JAVA",
			input:    storage.EmbeddedVirtualMachineScanComponent_JAVA,
			expected: v2.SourceType_JAVA,
		},
		{
			name:     "RUBY",
			input:    storage.EmbeddedVirtualMachineScanComponent_RUBY,
			expected: v2.SourceType_RUBY,
		},
		{
			name:     "NODEJS",
			input:    storage.EmbeddedVirtualMachineScanComponent_NODEJS,
			expected: v2.SourceType_NODEJS,
		},
		{
			name:     "GO",
			input:    storage.EmbeddedVirtualMachineScanComponent_GO,
			expected: v2.SourceType_GO,
		},
		{
			name:     "DotNetCoreRuntime",
			input:    storage.EmbeddedVirtualMachineScanComponent_DOTNETCORERUNTIME,
			expected: v2.SourceType_DOTNETCORERUNTIME,
		},
		{
			name:     "INFRASTRUCTURE",
			input:    storage.EmbeddedVirtualMachineScanComponent_INFRASTRUCTURE,
			expected: v2.SourceType_INFRASTRUCTURE,
		},
		{
			name:     "Other",
			input:    storage.EmbeddedVirtualMachineScanComponent_SourceType(-1),
			expected: v2.SourceType_OS,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertVirtualMachineSourceType(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVirtualMachineExecutables(t *testing.T) {
	tests := []struct {
		name     string
		input    []*storage.EmbeddedVirtualMachineScanComponent_Executable
		expected []*v2.ScanComponent_Executable
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "complete executable and nil",
			input: []*storage.EmbeddedVirtualMachineScanComponent_Executable{
				{
					Path:         "/usr/bin/test-exec",
					Dependencies: []string{"dependency1", "dependency2"},
				},
				nil,
			},
			expected: []*v2.ScanComponent_Executable{
				{
					Path:         "/usr/bin/test-exec",
					Dependencies: []string{"dependency1", "dependency2"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := VirtualMachineExecutables(tt.input)
			protoassert.SlicesEqual(t, tt.expected, result)
		})
	}
}

func TestVirtualMachineExecutable(t *testing.T) {
	tests := []struct {
		name     string
		input    *storage.EmbeddedVirtualMachineScanComponent_Executable
		expected *v2.ScanComponent_Executable
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "complete executable",
			input: &storage.EmbeddedVirtualMachineScanComponent_Executable{
				Path:         "/usr/bin/test-exec",
				Dependencies: []string{"dependency1", "dependency2"},
			},
			expected: &v2.ScanComponent_Executable{
				Path:         "/usr/bin/test-exec",
				Dependencies: []string{"dependency1", "dependency2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := VirtualMachineExecutable(tt.input)
			protoassert.Equal(t, tt.expected, result)
		})
	}
}

func TestVirtualMachineDataSource(t *testing.T) {
	tests := []struct {
		name     string
		input    *storage.VirtualMachineScan_DataSource
		expected *v2.DataSource
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "complete datasource",
			input: &storage.VirtualMachineScan_DataSource{
				Id:     "ds-123",
				Name:   "production-scanner",
				Mirror: "scanner.prod.example.com",
			},
			expected: &v2.DataSource{
				Id:     "ds-123",
				Name:   "production-scanner",
				Mirror: "scanner.prod.example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := VirtualMachineDataSource(tt.input)
			protoassert.Equal(t, tt.expected, result)
		})
	}
}
