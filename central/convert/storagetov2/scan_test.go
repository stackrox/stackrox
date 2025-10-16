package storagetov2

import (
	"testing"

	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
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
				storage.EmbeddedVirtualMachineScanComponent_builder{
					Name:      testComponentName,
					Version:   testComponentVersion,
					RiskScore: testComponentRiskScore,
					Vulnerabilities: []*storage.VirtualMachineVulnerability{
						storageVirtualMachineTestVuln,
					},
					Notes: []storage.EmbeddedVirtualMachineScanComponent_Note{
						storage.EmbeddedVirtualMachineScanComponent_UNSCANNED,
					},
				}.Build(),
			},
			expected: []*v2.ScanComponent{
				v2.ScanComponent_builder{
					Name:      testComponentName,
					Version:   testComponentVersion,
					RiskScore: testComponentRiskScore,
					Vulns: []*v2.EmbeddedVulnerability{
						v2VirtualMachineTestVuln,
					},
					Notes: []v2.ScanComponent_Note{
						v2.ScanComponent_UNSCANNED,
					},
				}.Build(),
			},
		},
		{
			name: "nil entries in input vector are ignored",
			input: []*storage.EmbeddedVirtualMachineScanComponent{
				nil,
				storage.EmbeddedVirtualMachineScanComponent_builder{
					Name:      testComponentName,
					Version:   testComponentVersion,
					RiskScore: testComponentRiskScore,
					Vulnerabilities: []*storage.VirtualMachineVulnerability{
						storageVirtualMachineTestVuln,
					},
				}.Build(),
				nil,
			},
			expected: []*v2.ScanComponent{
				v2.ScanComponent_builder{
					Name:      testComponentName,
					Version:   testComponentVersion,
					RiskScore: testComponentRiskScore,
					Vulns: []*v2.EmbeddedVulnerability{
						v2VirtualMachineTestVuln,
					},
				}.Build(),
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
			input: storage.EmbeddedVirtualMachineScanComponent_builder{
				Name:      testComponentName,
				Version:   testComponentVersion,
				RiskScore: testComponentRiskScore,
				Vulnerabilities: []*storage.VirtualMachineVulnerability{
					storageVirtualMachineTestVuln,
				},
				Source: storage.SourceType_INFRASTRUCTURE,
			}.Build(),
			expected: v2.ScanComponent_builder{
				Name:      testComponentName,
				Version:   testComponentVersion,
				RiskScore: testComponentRiskScore,
				Vulns: []*v2.EmbeddedVulnerability{
					v2VirtualMachineTestVuln,
				},
				Source: v2.SourceType_INFRASTRUCTURE,
			}.Build(),
		},
		{
			name: "component with SetTopCVSS",
			input: storage.EmbeddedVirtualMachineScanComponent_builder{
				Name:      testComponentName,
				Version:   testComponentVersion,
				RiskScore: testComponentRiskScore,
				TopCvss:   proto.Float32(7.1),
				Vulnerabilities: []*storage.VirtualMachineVulnerability{
					storageVirtualMachineTestVuln,
				},
			}.Build(),
			expected: v2.ScanComponent_builder{
				Name:      testComponentName,
				Version:   testComponentVersion,
				RiskScore: testComponentRiskScore,
				TopCvss:   proto.Float32(7.1),
				Vulns: []*v2.EmbeddedVulnerability{
					v2VirtualMachineTestVuln,
				},
			}.Build(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(it *testing.T) {
			result := EmbeddedVirtualMachineScanComponent(tc.input)
			protoassert.Equal(it, tc.expected, result)
		})
	}
}

func TestConvertSourceType(t *testing.T) {
	tests := map[string]struct {
		input    storage.SourceType
		expected v2.SourceType
	}{
		"OS": {
			input:    storage.SourceType_OS,
			expected: v2.SourceType_OS,
		},
		"PYTHON": {
			input:    storage.SourceType_PYTHON,
			expected: v2.SourceType_PYTHON,
		},
		"JAVA": {
			input:    storage.SourceType_JAVA,
			expected: v2.SourceType_JAVA,
		},
		"RUBY": {
			input:    storage.SourceType_RUBY,
			expected: v2.SourceType_RUBY,
		},
		"NODEJS": {
			input:    storage.SourceType_NODEJS,
			expected: v2.SourceType_NODEJS,
		},
		"GO": {
			input:    storage.SourceType_GO,
			expected: v2.SourceType_GO,
		},
		"DOTNETCORERUNTIME": {
			input:    storage.SourceType_DOTNETCORERUNTIME,
			expected: v2.SourceType_DOTNETCORERUNTIME,
		},
		"INFRASTRUCTURE": {
			input:    storage.SourceType_INFRASTRUCTURE,
			expected: v2.SourceType_INFRASTRUCTURE,
		},
		"Default": {
			input:    storage.SourceType(-1),
			expected: v2.SourceType_OS,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(it *testing.T) {
			result := convertSourceType(tc.input)
			assert.Equal(it, tc.expected, result)
		})
	}
}

func TestConvertScanComponentNoteType(t *testing.T) {
	tests := map[string]struct {
		input    storage.EmbeddedVirtualMachineScanComponent_Note
		expected v2.ScanComponent_Note
	}{
		"UNSPECIFIED": {
			input:    storage.EmbeddedVirtualMachineScanComponent_UNSPECIFIED,
			expected: v2.ScanComponent_UNSPECIFIED,
		},
		"UNSCANNED": {
			input:    storage.EmbeddedVirtualMachineScanComponent_UNSCANNED,
			expected: v2.ScanComponent_UNSCANNED,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(it *testing.T) {
			result := convertScanComponentNoteType(tc.input)
			assert.Equal(it, tc.expected, result)
		})
	}
}
