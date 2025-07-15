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

func TestDataSource(t *testing.T) {
	tests := []struct {
		name     string
		input    *v2.DataSource
		expected *storage.DataSource
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "complete datasource",
			input: &v2.DataSource{
				Id:     "ds-123",
				Name:   "production-scanner",
				Mirror: "scanner.prod.example.com",
			},
			expected: &storage.DataSource{
				Id:     "ds-123",
				Name:   "production-scanner",
				Mirror: "scanner.prod.example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DataSource(tt.input)
			protoassert.Equal(t, tt.expected, result)
		})
	}
}

func TestScanComponent(t *testing.T) {
	tests := []struct {
		name     string
		input    *v2.ScanComponent
		expected *storage.EmbeddedImageScanComponent
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "component with top cvss",
			input: &v2.ScanComponent{
				Name:    "vulnerable-lib",
				Version: "1.2.3",
				License: &v2.License{
					Name: "MIT",
					Type: "permissive",
					Url:  "https://opensource.org/licenses/MIT",
				},
				Source:       v2.SourceType_PYTHON,
				Location:     "/usr/lib/python/vulnerable-lib",
				SetTopCvss:   &v2.ScanComponent_TopCvss{TopCvss: 9.8},
				RiskScore:    8.5,
				FixedBy:      "1.2.4",
				Architecture: "amd64",
				Executables: []*v2.ScanComponent_Executable{
					{
						Path:         "/usr/bin/vuln-exec",
						Dependencies: []string{"lib1", "lib2"},
					},
				},
			},
			expected: &storage.EmbeddedImageScanComponent{
				Name:    "vulnerable-lib",
				Version: "1.2.3",
				License: &storage.License{
					Name: "MIT",
					Type: "permissive",
					Url:  "https://opensource.org/licenses/MIT",
				},
				Source:       storage.SourceType_PYTHON,
				Location:     "/usr/lib/python/vulnerable-lib",
				RiskScore:    8.5,
				FixedBy:      "1.2.4",
				Architecture: "amd64",
				SetTopCvss: &storage.EmbeddedImageScanComponent_TopCvss{
					TopCvss: 9.8,
				},
				Executables: []*storage.EmbeddedImageScanComponent_Executable{
					{
						Path:         "/usr/bin/vuln-exec",
						Dependencies: []string{"lib1", "lib2"},
					},
				},
			},
		},
		{
			name: "component without top cvss",
			input: &v2.ScanComponent{
				Name:    "safe-lib",
				Version: "2.0.0",
				Source:  v2.SourceType_GO,
			},
			expected: &storage.EmbeddedImageScanComponent{
				Name:    "safe-lib",
				Version: "2.0.0",
				Source:  storage.SourceType_GO,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ScanComponent(tt.input)
			protoassert.Equal(t, tt.expected, result)
		})
	}
}

func TestEmbeddedVulnerability(t *testing.T) {
	publishedOn := timestamppb.New(time.Now().Add(-24 * time.Hour))
	lastModified := timestamppb.New(time.Now())

	tests := []struct {
		name     string
		input    *v2.EmbeddedVulnerability
		expected *storage.EmbeddedVulnerability
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "vulnerability with fixed by",
			input: &v2.EmbeddedVulnerability{
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
			expected: &storage.EmbeddedVulnerability{
				Cve:     "CVE-2023-1234",
				Summary: "Test vulnerability",
				Link:    "https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2023-1234",
				SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
					FixedBy: "1.2.4",
				},
				VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
				Severity:          storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
				CvssV3: &storage.CVSSV3{
					Vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
					Score:  9.8,
				},
				PublishedOn:  publishedOn,
				LastModified: lastModified,
			},
		},
		{
			name: "vulnerability without fixed by",
			input: &v2.EmbeddedVulnerability{
				Cve:               "CVE-2023-5678",
				Summary:           "Another test vulnerability",
				VulnerabilityType: v2.VulnerabilityType_NODE_VULNERABILITY,
				Severity:          v2.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
			},
			expected: &storage.EmbeddedVulnerability{
				Cve:               "CVE-2023-5678",
				Summary:           "Another test vulnerability",
				VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
				Severity:          storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EmbeddedVulnerability(tt.input)
			protoassert.Equal(t, tt.expected, result)
		})
	}
}

func TestCvssV3(t *testing.T) {
	tests := []struct {
		name     string
		input    *v2.CVSSV3
		expected *storage.CVSSV3
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "complete cvss v3",
			input: &v2.CVSSV3{
				Vector:              "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
				ExploitabilityScore: 3.9,
				ImpactScore:         5.9,
				AttackVector:        v2.CVSSV3_ATTACK_NETWORK,
				AttackComplexity:    v2.CVSSV3_COMPLEXITY_LOW,
				PrivilegesRequired:  v2.CVSSV3_PRIVILEGE_NONE,
				UserInteraction:     v2.CVSSV3_UI_NONE,
				Scope:               v2.CVSSV3_UNCHANGED,
				Confidentiality:     v2.CVSSV3_IMPACT_HIGH,
				Integrity:           v2.CVSSV3_IMPACT_HIGH,
				Availability:        v2.CVSSV3_IMPACT_HIGH,
				Score:               9.8,
				Severity:            v2.CVSSV3_CRITICAL,
			},
			expected: &storage.CVSSV3{
				Vector:              "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
				ExploitabilityScore: 3.9,
				ImpactScore:         5.9,
				AttackVector:        storage.CVSSV3_ATTACK_NETWORK,
				AttackComplexity:    storage.CVSSV3_COMPLEXITY_LOW,
				PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_NONE,
				UserInteraction:     storage.CVSSV3_UI_NONE,
				Scope:               storage.CVSSV3_UNCHANGED,
				Confidentiality:     storage.CVSSV3_IMPACT_HIGH,
				Integrity:           storage.CVSSV3_IMPACT_HIGH,
				Availability:        storage.CVSSV3_IMPACT_HIGH,
				Score:               9.8,
				Severity:            storage.CVSSV3_CRITICAL,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CvssV3(tt.input)
			protoassert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertSourceType(t *testing.T) {
	tests := []struct {
		name     string
		input    v2.SourceType
		expected storage.SourceType
	}{
		{
			name:     "OS source type",
			input:    v2.SourceType_OS,
			expected: storage.SourceType_OS,
		},
		{
			name:     "Python source type",
			input:    v2.SourceType_PYTHON,
			expected: storage.SourceType_PYTHON,
		},
		{
			name:     "Java source type",
			input:    v2.SourceType_JAVA,
			expected: storage.SourceType_JAVA,
		},
		{
			name:     "Go source type",
			input:    v2.SourceType_GO,
			expected: storage.SourceType_GO,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertSourceType(tt.input)
			assert.Equal(t, tt.expected, result)
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

func TestScanComponents(t *testing.T) {
	tests := []struct {
		name     string
		input    []*v2.ScanComponent
		expected []*storage.EmbeddedImageScanComponent
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty input",
			input:    []*v2.ScanComponent{},
			expected: nil,
		},
		{
			name: "multiple components",
			input: []*v2.ScanComponent{
				{
					Name:    "component1",
					Version: "1.0.0",
					Source:  v2.SourceType_OS,
				},
				{
					Name:    "component2",
					Version: "2.0.0",
					Source:  v2.SourceType_PYTHON,
				},
			},
			expected: []*storage.EmbeddedImageScanComponent{
				{
					Name:    "component1",
					Version: "1.0.0",
					Source:  storage.SourceType_OS,
				},
				{
					Name:    "component2",
					Version: "2.0.0",
					Source:  storage.SourceType_PYTHON,
				},
			},
		},
		{
			name: "with nil component",
			input: []*v2.ScanComponent{
				{
					Name:    "component1",
					Version: "1.0.0",
				},
				nil,
				{
					Name:    "component2",
					Version: "2.0.0",
				},
			},
			expected: []*storage.EmbeddedImageScanComponent{
				{
					Name:    "component1",
					Version: "1.0.0",
				},
				{
					Name:    "component2",
					Version: "2.0.0",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ScanComponents(tt.input)
			protoassert.SlicesEqual(t, tt.expected, result)
		})
	}
}

func TestExecutables(t *testing.T) {
	tests := []struct {
		name     string
		input    []*v2.ScanComponent_Executable
		expected []*storage.EmbeddedImageScanComponent_Executable
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty input",
			input:    []*v2.ScanComponent_Executable{},
			expected: nil,
		},
		{
			name: "multiple executables",
			input: []*v2.ScanComponent_Executable{
				{
					Path:         "/usr/bin/exec1",
					Dependencies: []string{"lib1", "lib2"},
				},
				{
					Path:         "/usr/bin/exec2",
					Dependencies: []string{"lib3"},
				},
			},
			expected: []*storage.EmbeddedImageScanComponent_Executable{
				{
					Path:         "/usr/bin/exec1",
					Dependencies: []string{"lib1", "lib2"},
				},
				{
					Path:         "/usr/bin/exec2",
					Dependencies: []string{"lib3"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Executables(tt.input)
			protoassert.SlicesEqual(t, tt.expected, result)
		})
	}
}

func TestExecutable(t *testing.T) {
	tests := []struct {
		name     string
		input    *v2.ScanComponent_Executable
		expected *storage.EmbeddedImageScanComponent_Executable
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "complete executable",
			input: &v2.ScanComponent_Executable{
				Path:         "/usr/bin/test-exec",
				Dependencies: []string{"dependency1", "dependency2"},
			},
			expected: &storage.EmbeddedImageScanComponent_Executable{
				Path:         "/usr/bin/test-exec",
				Dependencies: []string{"dependency1", "dependency2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Executable(tt.input)
			protoassert.Equal(t, tt.expected, result)
		})
	}
}

func TestLicense(t *testing.T) {
	tests := []struct {
		name     string
		input    *v2.License
		expected *storage.License
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "complete license",
			input: &v2.License{
				Name: "Apache-2.0",
				Type: "permissive",
				Url:  "https://opensource.org/licenses/Apache-2.0",
			},
			expected: &storage.License{
				Name: "Apache-2.0",
				Type: "permissive",
				Url:  "https://opensource.org/licenses/Apache-2.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := License(tt.input)
			protoassert.Equal(t, tt.expected, result)
		})
	}
}

func TestEmbeddedVulnerabilities(t *testing.T) {
	tests := []struct {
		name     string
		input    []*v2.EmbeddedVulnerability
		expected []*storage.EmbeddedVulnerability
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty input",
			input:    []*v2.EmbeddedVulnerability{},
			expected: nil,
		},
		{
			name: "multiple vulnerabilities",
			input: []*v2.EmbeddedVulnerability{
				{
					Cve:     "CVE-2023-1234",
					Summary: "Test vulnerability 1",
				},
				{
					Cve:     "CVE-2023-5678",
					Summary: "Test vulnerability 2",
				},
			},
			expected: []*storage.EmbeddedVulnerability{
				{
					Cve:     "CVE-2023-1234",
					Summary: "Test vulnerability 1",
				},
				{
					Cve:     "CVE-2023-5678",
					Summary: "Test vulnerability 2",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EmbeddedVulnerabilities(tt.input)
			protoassert.SlicesEqual(t, tt.expected, result)
		})
	}
}
