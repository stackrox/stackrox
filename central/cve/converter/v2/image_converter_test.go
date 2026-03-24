package converter

import (
	"testing"

	store "github.com/stackrox/rox/central/cve/image/v2/datastore/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestToImageCVEV2(t *testing.T) {
	publishedOn := timestamppb.Now()
	createdAt := timestamppb.Now()
	firstOccurrence := timestamppb.Now()
	fixAvailableAt := timestamppb.Now()

	cve := &storage.NormalizedCVE{
		Id:           "cve-uuid-123",
		CveName:      "CVE-2021-44228",
		Source:       "NVD",
		Severity:     "CRITICAL",
		CvssV2:       7.5,
		CvssV3:       10.0,
		NvdCvssV3:    9.8,
		Summary:      "Log4j RCE vulnerability",
		Link:         "https://nvd.nist.gov/vuln/detail/CVE-2021-44228",
		PublishedOn:  publishedOn,
		AdvisoryName: "Red Hat Security Advisory",
		AdvisoryLink: "https://access.redhat.com/errata/RHSA-2021:5129",
		CreatedAt:    createdAt,
	}

	edge := &storage.NormalizedComponentCVEEdge{
		Id:                    "edge-uuid-456",
		ComponentId:           "component-sha256-abc",
		CveId:                 "cve-uuid-123",
		IsFixable:             true,
		FixedBy:               "2.17.1",
		State:                 "OBSERVED",
		FirstSystemOccurrence: firstOccurrence,
		FixAvailableAt:        fixAvailableAt,
	}

	converter := NewImageCVEConverter()
	result := converter.ToImageCVEV2(cve, edge)

	require.NotNil(t, result)

	// Verify composite ID.
	assert.Equal(t, "cve-uuid-123#component-sha256-abc", result.GetId())

	// Verify CVE base info.
	require.NotNil(t, result.GetCveBaseInfo())
	assert.Equal(t, "CVE-2021-44228", result.GetCveBaseInfo().GetCve())
	assert.Equal(t, "Log4j RCE vulnerability", result.GetCveBaseInfo().GetSummary())
	assert.Equal(t, "https://nvd.nist.gov/vuln/detail/CVE-2021-44228", result.GetCveBaseInfo().GetLink())
	assert.Equal(t, publishedOn, result.GetCveBaseInfo().GetPublishedOn())
	assert.Equal(t, createdAt, result.GetCveBaseInfo().GetCreatedAt())

	// Verify CVSS scores.
	assert.Equal(t, float32(10.0), result.GetCvss())
	assert.Equal(t, float32(9.8), result.GetNvdcvss())

	// Verify severity enum conversion.
	assert.Equal(t, storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY, result.GetSeverity())

	// Verify component and fixability.
	assert.Equal(t, "component-sha256-abc", result.GetComponentId())
	assert.True(t, result.GetIsFixable())
	assert.Equal(t, "2.17.1", result.GetFixedBy())

	// Verify state enum conversion.
	assert.Equal(t, storage.VulnerabilityState_OBSERVED, result.GetState())

	// Verify advisory.
	require.NotNil(t, result.GetAdvisory())
	assert.Equal(t, "Red Hat Security Advisory", result.GetAdvisory().GetName())
	assert.Equal(t, "https://access.redhat.com/errata/RHSA-2021:5129", result.GetAdvisory().GetLink())

	// Verify timestamps.
	assert.Equal(t, firstOccurrence, result.GetFirstImageOccurrence())
	assert.Equal(t, fixAvailableAt, result.GetFixAvailableTimestamp())

	// Verify datasource.
	assert.Equal(t, "NVD", result.GetDatasource())
}

func TestToImageCVEV2_NilInputs(t *testing.T) {
	converter := NewImageCVEConverter()

	// Test nil CVE.
	result := converter.ToImageCVEV2(nil, &storage.NormalizedComponentCVEEdge{})
	assert.Nil(t, result, "Should return nil when CVE is nil")

	// Test nil Edge.
	result = converter.ToImageCVEV2(&storage.NormalizedCVE{}, nil)
	assert.Nil(t, result, "Should return nil when Edge is nil")

	// Test both nil.
	result = converter.ToImageCVEV2(nil, nil)
	assert.Nil(t, result, "Should return nil when both are nil")
}

func TestToImageCVEV2_EmptyAdvisory(t *testing.T) {
	cve := &storage.NormalizedCVE{
		Id:       "cve-uuid-123",
		CveName:  "CVE-2021-44228",
		Severity: "HIGH",
		CvssV3:   8.5,
		// No advisory name or link.
	}

	edge := &storage.NormalizedComponentCVEEdge{
		ComponentId: "component-sha256-abc",
		CveId:       "cve-uuid-123",
		IsFixable:   false,
		State:       "DEFERRED",
	}

	converter := NewImageCVEConverter()
	result := converter.ToImageCVEV2(cve, edge)

	require.NotNil(t, result)
	assert.Nil(t, result.GetAdvisory(), "Advisory should be nil when no name or link")
	assert.Equal(t, storage.VulnerabilityState_DEFERRED, result.GetState())
	assert.Equal(t, storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY, result.GetSeverity())
}

func TestToImageCVEV2_SeverityMapping(t *testing.T) {
	testCases := []struct {
		name           string
		severityString string
		expectedEnum   storage.VulnerabilitySeverity
	}{
		{
			name:           "CRITICAL",
			severityString: "CRITICAL",
			expectedEnum:   storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
		},
		{
			name:           "HIGH",
			severityString: "HIGH",
			expectedEnum:   storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
		},
		{
			name:           "IMPORTANT",
			severityString: "IMPORTANT",
			expectedEnum:   storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
		},
		{
			name:           "MEDIUM",
			severityString: "MEDIUM",
			expectedEnum:   storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
		},
		{
			name:           "MODERATE",
			severityString: "MODERATE",
			expectedEnum:   storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
		},
		{
			name:           "LOW",
			severityString: "LOW",
			expectedEnum:   storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
		},
		{
			name:           "UNKNOWN",
			severityString: "UNKNOWN",
			expectedEnum:   storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY,
		},
		{
			name:           "Lowercase critical",
			severityString: "critical",
			expectedEnum:   storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
		},
		{
			name:           "Invalid severity",
			severityString: "INVALID",
			expectedEnum:   storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY,
		},
		{
			name:           "Empty severity",
			severityString: "",
			expectedEnum:   storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY,
		},
	}

	converter := NewImageCVEConverter()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cve := &storage.NormalizedCVE{
				Id:       "cve-uuid",
				CveName:  "CVE-2021-1234",
				Severity: tc.severityString,
			}
			edge := &storage.NormalizedComponentCVEEdge{
				ComponentId: "component-id",
				CveId:       "cve-uuid",
			}

			result := converter.ToImageCVEV2(cve, edge)
			require.NotNil(t, result)
			assert.Equal(t, tc.expectedEnum, result.GetSeverity())
		})
	}
}

func TestToImageCVEV2_StateMapping(t *testing.T) {
	testCases := []struct {
		name         string
		stateString  string
		expectedEnum storage.VulnerabilityState
	}{
		{
			name:         "OBSERVED",
			stateString:  "OBSERVED",
			expectedEnum: storage.VulnerabilityState_OBSERVED,
		},
		{
			name:         "DEFERRED",
			stateString:  "DEFERRED",
			expectedEnum: storage.VulnerabilityState_DEFERRED,
		},
		{
			name:         "FALSE_POSITIVE",
			stateString:  "FALSE_POSITIVE",
			expectedEnum: storage.VulnerabilityState_FALSE_POSITIVE,
		},
		{
			name:         "Lowercase observed",
			stateString:  "observed",
			expectedEnum: storage.VulnerabilityState_OBSERVED,
		},
		{
			name:         "Invalid state",
			stateString:  "INVALID",
			expectedEnum: storage.VulnerabilityState_OBSERVED,
		},
		{
			name:         "Empty state",
			stateString:  "",
			expectedEnum: storage.VulnerabilityState_OBSERVED,
		},
	}

	converter := NewImageCVEConverter()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cve := &storage.NormalizedCVE{
				Id:      "cve-uuid",
				CveName: "CVE-2021-1234",
			}
			edge := &storage.NormalizedComponentCVEEdge{
				ComponentId: "component-id",
				CveId:       "cve-uuid",
				State:       tc.stateString,
			}

			result := converter.ToImageCVEV2(cve, edge)
			require.NotNil(t, result)
			assert.Equal(t, tc.expectedEnum, result.GetState())
		})
	}
}

func TestToImageCVEV2Batch(t *testing.T) {
	publishedOn := timestamppb.Now()
	createdAt := timestamppb.Now()

	pairs := []store.CVEEdgePair{
		{
			CVE: &storage.NormalizedCVE{
				Id:          "cve-uuid-1",
				CveName:     "CVE-2021-1234",
				Severity:    "CRITICAL",
				CvssV3:      9.0,
				PublishedOn: publishedOn,
				CreatedAt:   createdAt,
			},
			Edge: &storage.NormalizedComponentCVEEdge{
				ComponentId: "component-1",
				CveId:       "cve-uuid-1",
				IsFixable:   true,
				FixedBy:     "1.0.0",
			},
		},
		{
			CVE: &storage.NormalizedCVE{
				Id:          "cve-uuid-2",
				CveName:     "CVE-2021-5678",
				Severity:    "HIGH",
				CvssV3:      7.5,
				PublishedOn: publishedOn,
				CreatedAt:   createdAt,
			},
			Edge: &storage.NormalizedComponentCVEEdge{
				ComponentId: "component-2",
				CveId:       "cve-uuid-2",
				IsFixable:   false,
			},
		},
		{
			CVE:  nil, // Nil CVE should be skipped.
			Edge: &storage.NormalizedComponentCVEEdge{},
		},
		{
			CVE:  &storage.NormalizedCVE{Id: "cve-uuid-3"},
			Edge: nil, // Nil Edge should be skipped.
		},
	}

	converter := NewImageCVEConverter()
	results := converter.ToImageCVEV2Batch(pairs)

	// Should only return 2 valid results (skip nils).
	require.Len(t, results, 2)

	// Verify first result.
	assert.Equal(t, "cve-uuid-1#component-1", results[0].GetId())
	assert.Equal(t, "CVE-2021-1234", results[0].GetCveBaseInfo().GetCve())
	assert.True(t, results[0].GetIsFixable())
	assert.Equal(t, "1.0.0", results[0].GetFixedBy())

	// Verify second result.
	assert.Equal(t, "cve-uuid-2#component-2", results[1].GetId())
	assert.Equal(t, "CVE-2021-5678", results[1].GetCveBaseInfo().GetCve())
	assert.False(t, results[1].GetIsFixable())
}

func TestToImageCVEV2Batch_EmptyInput(t *testing.T) {
	converter := NewImageCVEConverter()

	// Test empty slice.
	results := converter.ToImageCVEV2Batch(nil)
	assert.Empty(t, results, "Should return empty slice for nil input")

	results = converter.ToImageCVEV2Batch([]store.CVEEdgePair{})
	assert.Empty(t, results, "Should return empty slice for empty input")
}
