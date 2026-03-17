package v1tov2storage

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestScanPartsFromV1Scan_NilScan(t *testing.T) {
	parts := ScanPartsFromV1Scan("vm-1", nil)
	assert.Nil(t, parts.Scan)
	assert.Nil(t, parts.Components)
	assert.Nil(t, parts.CVEs)
	assert.Nil(t, parts.SourceComponents)
}

func TestScanPartsFromV1Scan_EmptyScan(t *testing.T) {
	scan := &storage.VirtualMachineScan{
		OperatingSystem: "rhel:9",
		ScanTime:        timestamppb.Now(),
	}
	parts := ScanPartsFromV1Scan("vm-1", scan)
	require.NotNil(t, parts.Scan)
	assert.Equal(t, "vm-1", parts.Scan.GetVmV2Id())
	assert.Equal(t, "rhel:9", parts.Scan.GetScanOs())
	assert.NotEmpty(t, parts.Scan.GetId())
	assert.Empty(t, parts.Components)
	assert.Empty(t, parts.CVEs)
}

func TestScanPartsFromV1Scan_WithComponentsAndVulns(t *testing.T) {
	scan := &storage.VirtualMachineScan{
		OperatingSystem: "rhel:9",
		ScanTime:        timestamppb.Now(),
		Components: []*storage.EmbeddedVirtualMachineScanComponent{
			{
				Name:    "openssl",
				Version: "3.0.7",
				Source:  storage.SourceType_OS,
				SetTopCvss: &storage.EmbeddedVirtualMachineScanComponent_TopCvss{
					TopCvss: 9.8,
				},
				Vulnerabilities: []*storage.VirtualMachineVulnerability{
					{
						CveBaseInfo: &storage.VirtualMachineCVEInfo{
							Cve:     "CVE-2023-0001",
							Summary: "test vuln",
							Link:    "https://example.com/CVE-2023-0001",
							CvssMetrics: []*storage.CVSSScore{
								{
									Source: storage.Source_SOURCE_NVD,
									CvssScore: &storage.CVSSScore_Cvssv3{
										Cvssv3: &storage.CVSSV3{
											Score:       9.8,
											ImpactScore: 5.9,
										},
									},
								},
							},
							Epss: &storage.VirtualMachineEPSS{
								EpssProbability: 0.5,
							},
							Advisory: &storage.VirtualMachineAdvisory{
								Name: "RHSA-2023:1234",
								Link: "https://example.com/RHSA-2023:1234",
							},
						},
						Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
						Cvss:     9.8,
						SetFixedBy: &storage.VirtualMachineVulnerability_FixedBy{
							FixedBy: "3.0.8",
						},
					},
					{
						CveBaseInfo: &storage.VirtualMachineCVEInfo{
							Cve:     "CVE-2023-0002",
							Summary: "test vuln 2",
							CvssMetrics: []*storage.CVSSScore{
								{
									Source: storage.Source_SOURCE_NVD,
									CvssScore: &storage.CVSSScore_Cvssv2{
										Cvssv2: &storage.CVSSV2{
											Score:       7.5,
											ImpactScore: 6.4,
										},
									},
								},
							},
						},
						Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
						Cvss:     7.5,
					},
				},
			},
		},
	}

	parts := ScanPartsFromV1Scan("vm-1", scan)

	// Scan
	require.NotNil(t, parts.Scan)
	assert.Equal(t, "vm-1", parts.Scan.GetVmV2Id())
	assert.Equal(t, "rhel:9", parts.Scan.GetScanOs())
	scanID := parts.Scan.GetId()
	assert.NotEmpty(t, scanID)

	// Components
	require.Len(t, parts.Components, 1)
	comp := parts.Components[0]
	assert.Equal(t, scanID, comp.GetVmScanId())
	assert.Equal(t, "openssl", comp.GetName())
	assert.Equal(t, "3.0.7", comp.GetVersion())
	assert.Equal(t, storage.SourceType_OS, comp.GetSource())
	assert.InDelta(t, 9.8, float64(comp.GetTopCvss()), 0.01)
	assert.Equal(t, "3.0.8", comp.GetFixedBy())
	assert.Equal(t, int32(2), comp.GetCveCount())

	// CVEs
	require.Len(t, parts.CVEs, 2)

	cve1 := parts.CVEs[0]
	assert.Equal(t, "vm-1", cve1.GetVmV2Id())
	assert.Equal(t, comp.GetId(), cve1.GetVmComponentId())
	assert.Equal(t, "CVE-2023-0001", cve1.GetCveBaseInfo().GetCve())
	assert.InDelta(t, 9.8, float64(cve1.GetPreferredCvss()), 0.01)
	assert.Equal(t, storage.CvssScoreVersion_V3, cve1.GetPreferredCvssVersion())
	assert.Equal(t, storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY, cve1.GetSeverity())
	assert.InDelta(t, 5.9, float64(cve1.GetImpactScore()), 0.01)
	assert.InDelta(t, 9.8, float64(cve1.GetNvdcvss()), 0.01)
	assert.Equal(t, storage.CvssScoreVersion_V3, cve1.GetNvdScoreVersion())
	assert.True(t, cve1.GetIsFixable())
	assert.Equal(t, "3.0.8", cve1.GetFixedBy())
	assert.InDelta(t, 0.5, float64(cve1.GetEpssProbability()), 0.01)
	assert.Equal(t, "RHSA-2023:1234", cve1.GetAdvisory().GetName())

	cve2 := parts.CVEs[1]
	assert.Equal(t, "CVE-2023-0002", cve2.GetCveBaseInfo().GetCve())
	assert.Equal(t, storage.CvssScoreVersion_V2, cve2.GetPreferredCvssVersion())
	assert.InDelta(t, 7.5, float64(cve2.GetNvdcvss()), 0.01)
	assert.Equal(t, storage.CvssScoreVersion_V2, cve2.GetNvdScoreVersion())
	assert.False(t, cve2.GetIsFixable())
	assert.Empty(t, cve2.GetFixedBy())

	// SourceComponents preserved
	protoassert.SlicesEqual(t, scan.GetComponents(), parts.SourceComponents)
}

func TestHighestFixedBy(t *testing.T) {
	tests := []struct {
		name     string
		vulns    []*storage.VirtualMachineVulnerability
		expected string
	}{
		{
			name:     "nil",
			expected: "",
		},
		{
			name: "no fixable",
			vulns: []*storage.VirtualMachineVulnerability{
				{Cvss: 5.0},
			},
			expected: "",
		},
		{
			name: "single fixable",
			vulns: []*storage.VirtualMachineVulnerability{
				{SetFixedBy: &storage.VirtualMachineVulnerability_FixedBy{FixedBy: "1.2.3"}},
			},
			expected: "1.2.3",
		},
		{
			name: "highest picked",
			vulns: []*storage.VirtualMachineVulnerability{
				{SetFixedBy: &storage.VirtualMachineVulnerability_FixedBy{FixedBy: "1.0.0"}},
				{SetFixedBy: &storage.VirtualMachineVulnerability_FixedBy{FixedBy: "2.0.0"}},
				{SetFixedBy: &storage.VirtualMachineVulnerability_FixedBy{FixedBy: "1.5.0"}},
			},
			expected: "2.0.0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, highestFixedBy(tt.vulns))
		})
	}
}
