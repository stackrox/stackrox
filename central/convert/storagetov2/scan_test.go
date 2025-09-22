package storagetov2

import (
	"testing"

	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
)

const (
	testComponentName      = "wordpress-ns-simple-intro-loader"
	testComponentVersion   = "2.2.3"
	testComponentRiskScore = 4.5
)

func TestEmbeddedVirtualMachineScanComponents(t *testing.T) {
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
			name: "input vector without nil entry",
			input: []*storage.EmbeddedVirtualMachineScanComponent{
				{
					Name:      testComponentName,
					Version:   testComponentVersion,
					RiskScore: testComponentRiskScore,
					Vulnerabilities: []*storage.VirtualMachineVulnerability{
						storageVirtualMachineTestVuln,
					},
				},
			},
			expected: []*v2.ScanComponent{
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
		{
			name: "nil entries in input vector are ignored",
			input: []*storage.EmbeddedVirtualMachineScanComponent{
				nil,
				{
					Name:      testComponentName,
					Version:   testComponentVersion,
					RiskScore: testComponentRiskScore,
					Vulnerabilities: []*storage.VirtualMachineVulnerability{
						storageVirtualMachineTestVuln,
					},
				},
				nil,
			},
			expected: []*v2.ScanComponent{
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
	}

	for _, tc := range tests {
		t.Run(tc.name, func(it *testing.T) {
			result := EmbeddedVirtualMachineScanComponents(tc.input)
			protoassert.SlicesEqual(it, tc.expected, result)
		})
	}
}

func TestEmbeddedVirtualMachineScanComponent(t *testing.T) {
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
			name: "component without SetTopCVSS",
			input: &storage.EmbeddedVirtualMachineScanComponent{
				Name:      testComponentName,
				Version:   testComponentVersion,
				RiskScore: testComponentRiskScore,
				Vulnerabilities: []*storage.VirtualMachineVulnerability{
					storageVirtualMachineTestVuln,
				},
			},
			expected: &v2.ScanComponent{
				Name:      testComponentName,
				Version:   testComponentVersion,
				RiskScore: testComponentRiskScore,
				Vulns: []*v2.EmbeddedVulnerability{
					v2VirtualMachineTestVuln,
				},
			},
		},
		{
			name: "component with SetTopCVSS",
			input: &storage.EmbeddedVirtualMachineScanComponent{
				Name:      testComponentName,
				Version:   testComponentVersion,
				RiskScore: testComponentRiskScore,
				SetTopCvss: &storage.EmbeddedVirtualMachineScanComponent_TopCvss{
					TopCvss: 7.1,
				},
				Vulnerabilities: []*storage.VirtualMachineVulnerability{
					storageVirtualMachineTestVuln,
				},
			},
			expected: &v2.ScanComponent{
				Name:      testComponentName,
				Version:   testComponentVersion,
				RiskScore: testComponentRiskScore,
				SetTopCvss: &v2.ScanComponent_TopCvss{
					TopCvss: 7.1,
				},
				Vulns: []*v2.EmbeddedVulnerability{
					v2VirtualMachineTestVuln,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(it *testing.T) {
			result := EmbeddedVirtualMachineScanComponent(tc.input)
			protoassert.Equal(it, tc.expected, result)
		})
	}
}

func TestScanComponent(t *testing.T) {
	tests := []struct {
		name     string
		input    *storage.EmbeddedImageScanComponent
		expected *v2.ScanComponent
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "component with top cvss",
			input: &storage.EmbeddedImageScanComponent{
				Name:         "vulnerable-lib",
				Version:      "1.2.3",
				RiskScore:    8.5,
				Architecture: "amd64",
				SetTopCvss: &storage.EmbeddedImageScanComponent_TopCvss{
					TopCvss: 9.8,
				},
			},
			expected: &v2.ScanComponent{
				Name:         "vulnerable-lib",
				Version:      "1.2.3",
				RiskScore:    8.5,
				Architecture: "amd64",
				SetTopCvss: &v2.ScanComponent_TopCvss{
					TopCvss: 9.8,
				},
			},
		},
		{
			name: "component without top cvss",
			input: &storage.EmbeddedImageScanComponent{
				Name:    "safe-lib",
				Version: "2.0.0",
			},
			expected: &v2.ScanComponent{
				Name:    "safe-lib",
				Version: "2.0.0",
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

func TestScanComponents(t *testing.T) {
	tests := []struct {
		name     string
		input    []*storage.EmbeddedImageScanComponent
		expected []*v2.ScanComponent
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty input",
			input:    []*storage.EmbeddedImageScanComponent{},
			expected: nil,
		},
		{
			name: "multiple components",
			input: []*storage.EmbeddedImageScanComponent{
				{
					Name:    "component1",
					Version: "1.0.0",
				},
				{
					Name:    "component2",
					Version: "2.0.0",
				},
			},
			expected: []*v2.ScanComponent{
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
		{
			name: "with nil component",
			input: []*storage.EmbeddedImageScanComponent{
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
			expected: []*v2.ScanComponent{
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
