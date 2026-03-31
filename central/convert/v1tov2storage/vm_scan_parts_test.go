package v1tov2storage

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanPartsFromV1Scan_NilScan(t *testing.T) {
	result := ScanPartsFromV1Scan("vm-1", nil)
	assert.Nil(t, result)
}

func TestScanPartsFromV1Scan_EmptyScan(t *testing.T) {
	result := ScanPartsFromV1Scan("vm-1", &storage.VirtualMachineScan{})
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Scan.GetId())
	assert.Equal(t, "vm-1", result.Scan.GetVmV2Id())
	assert.Empty(t, result.Components)
	assert.Empty(t, result.CVEs)
}

func TestScanPartsFromV1Scan_WithComponentsAndVulns(t *testing.T) {
	scan := &storage.VirtualMachineScan{
		OperatingSystem: "rhel:9",
		Notes:           []storage.VirtualMachineScan_Note{storage.VirtualMachineScan_OS_UNKNOWN},
		Components: []*storage.EmbeddedVirtualMachineScanComponent{
			{
				Name:    "openssl",
				Version: "1.1.1",
				Source:  storage.SourceType_OS,
				Vulnerabilities: []*storage.VirtualMachineVulnerability{
					{
						CveBaseInfo: &storage.VirtualMachineCVEInfo{
							Cve:     "CVE-2023-0001",
							Summary: "test vuln",
							CvssMetrics: []*storage.CVSSScore{
								{
									Source: storage.Source_SOURCE_RED_HAT,
									CvssScore: &storage.CVSSScore_Cvssv3{
										Cvssv3: &storage.CVSSV3{
											Score:       7.5,
											ImpactScore: 3.6,
										},
									},
								},
								{
									Source: storage.Source_SOURCE_NVD,
									CvssScore: &storage.CVSSScore_Cvssv3{
										Cvssv3: &storage.CVSSV3{
											Score:       8.0,
											ImpactScore: 4.0,
										},
									},
								},
							},
							Epss: &storage.VirtualMachineEPSS{
								EpssProbability: 0.5,
							},
							Advisory: &storage.VirtualMachineAdvisory{
								Name: "RHSA-2023:1234",
								Link: "https://example.com/advisory",
							},
						},
						Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
						SetFixedBy: &storage.VirtualMachineVulnerability_FixedBy{
							FixedBy: "1.1.2",
						},
						Cvss: 7.5,
					},
				},
			},
		},
	}

	result := ScanPartsFromV1Scan("vm-1", scan)
	require.NotNil(t, result)

	// Scan
	assert.Equal(t, "vm-1", result.Scan.GetVmV2Id())
	assert.Equal(t, "rhel:9", result.Scan.GetScanOs())
	assert.InDelta(t, float32(7.5), result.Scan.GetTopCvss(), 0.01)
	assert.Equal(t, []storage.VirtualMachineScanV2_Note{storage.VirtualMachineScanV2_OS_UNKNOWN}, result.Scan.GetNotes())

	// Components
	require.Len(t, result.Components, 1)
	comp := result.Components[0]
	assert.Equal(t, "openssl", comp.GetName())
	assert.Equal(t, "1.1.1", comp.GetVersion())
	assert.Equal(t, storage.SourceType_OS, comp.GetSource())
	assert.Equal(t, "rhel:9", comp.GetOperatingSystem())
	assert.Equal(t, "1.1.2", comp.GetFixedBy())
	assert.Equal(t, int32(1), comp.GetCveCount())

	// CVEs
	require.Len(t, result.CVEs, 1)
	cve := result.CVEs[0]
	assert.Equal(t, "vm-1", cve.GetVmV2Id())
	assert.Equal(t, comp.GetId(), cve.GetVmComponentId())
	assert.Equal(t, "CVE-2023-0001", cve.GetCveBaseInfo().GetCve())
	assert.InDelta(t, float32(7.5), cve.GetPreferredCvss(), 0.01)
	assert.Equal(t, storage.CvssScoreVersion_V3, cve.GetPreferredCvssVersion())
	assert.Equal(t, storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY, cve.GetSeverity())
	assert.InDelta(t, float32(3.6), cve.GetImpactScore(), 0.01)
	assert.InDelta(t, float32(8.0), cve.GetNvdcvss(), 0.01)
	assert.Equal(t, storage.CvssScoreVersion_V3, cve.GetNvdScoreVersion())
	assert.True(t, cve.GetIsFixable())
	assert.Equal(t, "1.1.2", cve.GetFixedBy())
	assert.InDelta(t, float32(0.5), cve.GetEpssProbability(), 0.01)
	assert.Equal(t, "RHSA-2023:1234", cve.GetAdvisory().GetName())

	// Source components preserved
	require.Len(t, result.SourceComponents, 1)
	assert.Equal(t, "openssl", result.SourceComponents[0].GetName())
}

func TestCompareVersionSegments(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want int
	}{
		{name: "equal", a: "1.2.3", b: "1.2.3", want: 0},
		{name: "a greater", a: "1.2.4", b: "1.2.3", want: 1},
		{name: "b greater", a: "1.2.3", b: "1.2.4", want: -1},
		{name: "different lengths a longer", a: "1.2.3.1", b: "1.2.3", want: 1},
		{name: "different lengths b longer", a: "1.2", b: "1.2.1", want: -1},
		{name: "major version diff", a: "2.0.0", b: "1.9.9", want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(it *testing.T) {
			got := compareVersionSegments(tt.a, tt.b)
			if tt.want > 0 {
				assert.Greater(it, got, 0)
			} else if tt.want < 0 {
				assert.Less(it, got, 0)
			} else {
				assert.Equal(it, 0, got)
			}
		})
	}
}

func TestHighestFixedBy(t *testing.T) {
	tests := []struct {
		name  string
		vulns []*storage.VirtualMachineVulnerability
		want  string
	}{
		{
			name:  "no vulns",
			vulns: nil,
			want:  "",
		},
		{
			name: "no fixed by",
			vulns: []*storage.VirtualMachineVulnerability{
				{CveBaseInfo: &storage.VirtualMachineCVEInfo{Cve: "CVE-1"}},
			},
			want: "",
		},
		{
			name: "single fixed by",
			vulns: []*storage.VirtualMachineVulnerability{
				{SetFixedBy: &storage.VirtualMachineVulnerability_FixedBy{FixedBy: "1.0.1"}},
			},
			want: "1.0.1",
		},
		{
			name: "picks highest",
			vulns: []*storage.VirtualMachineVulnerability{
				{SetFixedBy: &storage.VirtualMachineVulnerability_FixedBy{FixedBy: "1.0.1"}},
				{SetFixedBy: &storage.VirtualMachineVulnerability_FixedBy{FixedBy: "2.0.0"}},
				{SetFixedBy: &storage.VirtualMachineVulnerability_FixedBy{FixedBy: "1.5.3"}},
			},
			want: "2.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(it *testing.T) {
			got := highestFixedBy(tt.vulns)
			assert.Equal(it, tt.want, got)
		})
	}
}
