package service

import (
	"testing"

	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestCVEAccumulator_SingleRow(t *testing.T) {
	cve := &storage.ImageCVEV2{
		CveBaseInfo: &storage.CVEInfo{
			Cve:         "CVE-2024-1234",
			Summary:     "A test vulnerability.",
			Link:        "https://nvd.nist.gov/vuln/detail/CVE-2024-1234",
			PublishedOn: timestamppb.Now(),
			CvssMetrics: []*storage.CVSSScore{
				{
					Source: storage.Source_SOURCE_NVD,
					Url:    "https://nvd.nist.gov",
					CvssScore: &storage.CVSSScore_Cvssv3{
						Cvssv3: &storage.CVSSV3{Score: 9.1, Severity: storage.CVSSV3_CRITICAL},
					},
				},
			},
			Epss: &storage.EPSS{EpssProbability: 0.85, EpssPercentile: 0.95},
		},
		Severity:         storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
		Cvss:             9.1,
		ComponentName:    "openssl",
		ComponentVersion: "1.1.1k",
		RepositoryCpe:    "cpe:2.3:o:redhat:enterprise_linux:8:*:*:*:*:*:*:*",
		Advisory: &storage.Advisory{
			Name: "RHSA-2024:1234",
			Link: "https://access.redhat.com/errata/RHSA-2024:1234",
		},
	}

	acc := newCVEAccumulator(cve)
	detail := acc.toCVEDetail()

	assert.Equal(t, "CVE-2024-1234", detail.GetCve())
	assert.Equal(t, v2.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY, detail.GetSeverity())
	assert.InDelta(t, 9.1, detail.GetCvss(), 0.01)
	assert.Equal(t, "A test vulnerability.", detail.GetSummary())
	assert.InDelta(t, 0.85, detail.GetEpssProbability(), 0.01)
	assert.InDelta(t, 0.95, detail.GetEpssPercentile(), 0.01)
	assert.Equal(t, "RHSA-2024:1234", detail.GetAdvisory().GetName())
	assert.Len(t, detail.GetCvssScores(), 1)
	assert.Equal(t, v2.Source_SOURCE_NVD, detail.GetCvssScores()[0].GetSource())
	assert.Empty(t, detail.GetComponentOverrides())
}

func TestCVEAccumulator_MergeMaxSeverity(t *testing.T) {
	row1 := &storage.ImageCVEV2{
		CveBaseInfo: &storage.CVEInfo{
			Cve:     "CVE-2024-5678",
			Summary: "A vulnerability with multiple severity levels.",
			CvssMetrics: []*storage.CVSSScore{
				{
					Source:    storage.Source_SOURCE_RED_HAT,
					CvssScore: &storage.CVSSScore_Cvssv3{Cvssv3: &storage.CVSSV3{Score: 6.1}},
				},
			},
		},
		Severity:         storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
		Cvss:             6.1,
		ComponentName:    "curl",
		ComponentVersion: "7.68.0",
		RepositoryCpe:    "cpe:2.3:o:redhat:enterprise_linux:8:*:*:*:*:*:*:*",
	}
	row2 := &storage.ImageCVEV2{
		CveBaseInfo: &storage.CVEInfo{
			Cve: "CVE-2024-5678",
			CvssMetrics: []*storage.CVSSScore{
				{
					Source:    storage.Source_SOURCE_NVD,
					CvssScore: &storage.CVSSScore_Cvssv3{Cvssv3: &storage.CVSSV3{Score: 9.8}},
				},
			},
		},
		Severity:         storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
		Cvss:             9.8,
		ComponentName:    "curl",
		ComponentVersion: "7.68.0",
	}

	acc := newCVEAccumulator(row1)
	acc.merge(row2)
	detail := acc.toCVEDetail()

	assert.Equal(t, v2.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY, detail.GetSeverity())
	assert.InDelta(t, 9.8, detail.GetCvss(), 0.01)
	assert.Len(t, detail.GetCvssScores(), 2)
}

func TestCVEAccumulator_ComponentOverrides(t *testing.T) {
	row1 := &storage.ImageCVEV2{
		CveBaseInfo: &storage.CVEInfo{
			Cve: "CVE-2024-9999",
			CvssMetrics: []*storage.CVSSScore{
				{
					Source:    storage.Source_SOURCE_NVD,
					CvssScore: &storage.CVSSScore_Cvssv3{Cvssv3: &storage.CVSSV3{Score: 9.1}},
				},
			},
		},
		Severity:         storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
		Cvss:             9.1,
		ComponentName:    "openssl",
		ComponentVersion: "1.1.1k",
	}
	row2 := &storage.ImageCVEV2{
		CveBaseInfo: &storage.CVEInfo{
			Cve: "CVE-2024-9999",
			CvssMetrics: []*storage.CVSSScore{
				{
					Source:    storage.Source_SOURCE_RED_HAT,
					CvssScore: &storage.CVSSScore_Cvssv3{Cvssv3: &storage.CVSSV3{Score: 6.1}},
				},
			},
		},
		Severity:         storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
		Cvss:             6.1,
		ComponentName:    "subscription-manager",
		ComponentVersion: "1.28.29",
		RepositoryCpe:    "cpe:2.3:o:redhat:enterprise_linux:7:*:*:*:*:*:*:*",
	}

	acc := newCVEAccumulator(row1)
	acc.merge(row2)
	detail := acc.toCVEDetail()

	assert.Equal(t, v2.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY, detail.GetSeverity())
	assert.InDelta(t, 9.1, detail.GetCvss(), 0.01)

	assert.Len(t, detail.GetComponentOverrides(), 1)
	override := detail.GetComponentOverrides()[0]
	assert.Equal(t, "subscription-manager", override.GetComponentName())
	assert.Equal(t, "1.28.29", override.GetComponentVersion())
	assert.Equal(t, "cpe:2.3:o:redhat:enterprise_linux:7:*:*:*:*:*:*:*", override.GetRepositoryCpe())
	assert.Equal(t, v2.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY, override.GetSeverity())
	assert.InDelta(t, 6.1, override.GetCvss(), 0.01)
}

func TestCVEAccumulator_SameSeverityNoOverride(t *testing.T) {
	row1 := &storage.ImageCVEV2{
		CveBaseInfo: &storage.CVEInfo{Cve: "CVE-2024-0001"},
		Severity:    storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
		Cvss:        7.5,
	}
	row2 := &storage.ImageCVEV2{
		CveBaseInfo: &storage.CVEInfo{Cve: "CVE-2024-0001"},
		Severity:    storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
		Cvss:        7.5,
	}

	acc := newCVEAccumulator(row1)
	acc.merge(row2)
	detail := acc.toCVEDetail()

	assert.Empty(t, detail.GetComponentOverrides())
}

func TestCVEAccumulator_AdvisoryFromLaterRow(t *testing.T) {
	row1 := &storage.ImageCVEV2{
		CveBaseInfo: &storage.CVEInfo{Cve: "CVE-2024-0002"},
		Severity:    storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
		Cvss:        3.0,
	}
	row2 := &storage.ImageCVEV2{
		CveBaseInfo: &storage.CVEInfo{Cve: "CVE-2024-0002"},
		Severity:    storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
		Cvss:        3.0,
		Advisory: &storage.Advisory{
			Name: "RHSA-2024:0002",
			Link: "https://access.redhat.com/errata/RHSA-2024:0002",
		},
	}

	acc := newCVEAccumulator(row1)
	acc.merge(row2)
	detail := acc.toCVEDetail()

	assert.Equal(t, "RHSA-2024:0002", detail.GetAdvisory().GetName())
}

func TestCVEAccumulator_CVSSMetricsDedup(t *testing.T) {
	row1 := &storage.ImageCVEV2{
		CveBaseInfo: &storage.CVEInfo{
			Cve: "CVE-2024-0003",
			CvssMetrics: []*storage.CVSSScore{
				{
					Source:    storage.Source_SOURCE_RED_HAT,
					CvssScore: &storage.CVSSScore_Cvssv3{Cvssv3: &storage.CVSSV3{Score: 5.0}},
				},
				{
					Source:    storage.Source_SOURCE_NVD,
					CvssScore: &storage.CVSSScore_Cvssv3{Cvssv3: &storage.CVSSV3{Score: 7.0}},
				},
			},
		},
		Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
		Cvss:     7.0,
	}
	row2 := &storage.ImageCVEV2{
		CveBaseInfo: &storage.CVEInfo{
			Cve: "CVE-2024-0003",
			CvssMetrics: []*storage.CVSSScore{
				{
					Source:    storage.Source_SOURCE_RED_HAT,
					CvssScore: &storage.CVSSScore_Cvssv3{Cvssv3: &storage.CVSSV3{Score: 8.0}},
				},
				{
					Source:    storage.Source_SOURCE_NVD,
					CvssScore: &storage.CVSSScore_Cvssv3{Cvssv3: &storage.CVSSV3{Score: 7.0}},
				},
			},
		},
		Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
		Cvss:     7.0,
	}

	acc := newCVEAccumulator(row1)
	acc.merge(row2)
	detail := acc.toCVEDetail()

	assert.Len(t, detail.GetCvssScores(), 2)
	for _, score := range detail.GetCvssScores() {
		if score.GetSource() == v2.Source_SOURCE_RED_HAT {
			assert.InDelta(t, 8.0, score.GetCvssv3().GetScore(), 0.01, "Red Hat score should be the higher value.")
		}
		if score.GetSource() == v2.Source_SOURCE_NVD {
			assert.InDelta(t, 7.0, score.GetCvssv3().GetScore(), 0.01)
		}
	}
}

func TestCVEAccumulator_FirstRowLowerSeverity(t *testing.T) {
	row1 := &storage.ImageCVEV2{
		CveBaseInfo:      &storage.CVEInfo{Cve: "CVE-2024-LOW-FIRST"},
		Severity:         storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
		Cvss:             6.1,
		ComponentName:    "subscription-manager",
		ComponentVersion: "1.28.29",
		RepositoryCpe:    "cpe:2.3:o:redhat:enterprise_linux:7:*:*:*:*:*:*:*",
	}
	row2 := &storage.ImageCVEV2{
		CveBaseInfo:      &storage.CVEInfo{Cve: "CVE-2024-LOW-FIRST"},
		Severity:         storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
		Cvss:             9.8,
		ComponentName:    "openssl",
		ComponentVersion: "1.1.1k",
	}

	acc := newCVEAccumulator(row1)
	acc.merge(row2)
	detail := acc.toCVEDetail()

	assert.Equal(t, v2.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY, detail.GetSeverity())
	assert.InDelta(t, 9.8, detail.GetCvss(), 0.01)

	// The first row (MODERATE) should appear as an override since it differs from the final max.
	assert.Len(t, detail.GetComponentOverrides(), 1)
	override := detail.GetComponentOverrides()[0]
	assert.Equal(t, "subscription-manager", override.GetComponentName())
	assert.Equal(t, v2.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY, override.GetSeverity())
	assert.InDelta(t, 6.1, override.GetCvss(), 0.01)
}

func TestConvertSeverity(t *testing.T) {
	tests := map[string]struct {
		input    storage.VulnerabilitySeverity
		expected v2.VulnerabilitySeverity
	}{
		"low":       {storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY, v2.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY},
		"moderate":  {storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY, v2.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY},
		"important": {storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY, v2.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY},
		"critical":  {storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY, v2.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY},
		"unknown":   {storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY, v2.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, convertSeverity(tc.input))
		})
	}
}
