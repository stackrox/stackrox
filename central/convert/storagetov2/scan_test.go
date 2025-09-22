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
