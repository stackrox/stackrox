package enricher

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stackrox/rox/clair-adapter/clairclient"
	"github.com/stackrox/rox/clair-adapter/enricher/csaf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipeline_Enrich(t *testing.T) {
	tests := map[string]struct {
		vr       *clairclient.VulnerabilityReport
		pipeline *Pipeline
		validate func(t *testing.T, result *EnrichmentResult, err error)
	}{
		"vuln with fixed version populates PkgFixedBy": {
			vr: &clairclient.VulnerabilityReport{
				Packages: map[string]clairclient.Package{
					"pkg1": {ID: "pkg1", Name: "bash", Version: "5.0"},
				},
				Vulnerabilities: map[string]clairclient.Vulnerability{
					"vuln1": {ID: "vuln1", FixedInVersion: "5.2"},
				},
				PackageVulnerabilities: map[string][]string{
					"pkg1": {"vuln1"},
				},
			},
			pipeline: NewPipeline(),
			validate: func(t *testing.T, result *EnrichmentResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, map[string]string{"pkg1": "5.2"}, result.PkgFixedBy)
			},
		},
		"NVD enrichment data is extracted": {
			vr: &clairclient.VulnerabilityReport{
				Enrichments: map[string][]json.RawMessage{
					"message/vnd.clair.map.vulnerability; enricher=clair.cvss": {
						json.RawMessage(`{"CVE-2023-1234": [{"cve": "CVE-2023-1234", "cvss": [{"version": "3.1", "score": 7.5, "vector": "CVSS:3.1/AV:N/AC:L"}]}]}`),
					},
				},
			},
			pipeline: NewPipeline(),
			validate: func(t *testing.T, result *EnrichmentResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result.NVDVulns)
				assert.Len(t, result.NVDVulns, 1)
				for _, nvdMap := range result.NVDVulns {
					assert.Contains(t, nvdMap, "CVE-2023-1234")
					assert.Equal(t, 7.5, nvdMap["CVE-2023-1234"].CVSSv3.BaseScore)
				}
			},
		},
		"EPSS enrichment data is extracted": {
			vr: &clairclient.VulnerabilityReport{
				Enrichments: map[string][]json.RawMessage{
					"message/vnd.clair.map.enrichment; enricher=clair.epss": {
						json.RawMessage(`{"CVE-2023-1234": [{"cve": "CVE-2023-1234", "epss": {"model_version": "v2023.03.01", "date": "2023-05-15", "probability": 0.00123, "percentile": 0.45678}}]}`),
					},
				},
			},
			pipeline: NewPipeline(),
			validate: func(t *testing.T, result *EnrichmentResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result.EPSSItems)
				assert.Len(t, result.EPSSItems, 1)
				for _, epssMap := range result.EPSSItems {
					assert.Contains(t, epssMap, "CVE-2023-1234")
					assert.Equal(t, 0.00123, epssMap["CVE-2023-1234"].Probability)
				}
			},
		},
		"CSAF enrichment when enricher provided": {
			vr: &clairclient.VulnerabilityReport{
				Vulnerabilities: map[string]clairclient.Vulnerability{
					"vuln1": {ID: "vuln1", Name: "RHSA-2023:1234: bash update"},
				},
			},
			pipeline: NewPipeline(WithCSAFEnricher(csaf.NewEnricher(csaf.WithStaticAdvisories(map[string]*csaf.Advisory{
				"RHSA-2023:1234": {
					Name:        "RHSA-2023:1234",
					Description: "bash security update",
					Severity:    "Important",
				},
			})))),
			validate: func(t *testing.T, result *EnrichmentResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result.CSAFAdvisories)
				assert.Len(t, result.CSAFAdvisories, 1)
				assert.Contains(t, result.CSAFAdvisories, "vuln1")
				assert.Equal(t, "bash security update", result.CSAFAdvisories["vuln1"].Description)
			},
		},
		"no CSAF enrichment when enricher not provided": {
			vr: &clairclient.VulnerabilityReport{
				Vulnerabilities: map[string]clairclient.Vulnerability{
					"vuln1": {ID: "vuln1", Name: "RHSA-2023:1234: bash update"},
				},
			},
			pipeline: NewPipeline(),
			validate: func(t *testing.T, result *EnrichmentResult, err error) {
				require.NoError(t, err)
				assert.Empty(t, result.CSAFAdvisories)
			},
		},
		"all enrichments combined": {
			vr: &clairclient.VulnerabilityReport{
				Packages: map[string]clairclient.Package{
					"pkg1": {ID: "pkg1"},
				},
				Vulnerabilities: map[string]clairclient.Vulnerability{
					"vuln1": {ID: "vuln1", Name: "RHSA-2023:1234: test", FixedInVersion: "1.0"},
				},
				PackageVulnerabilities: map[string][]string{
					"pkg1": {"vuln1"},
				},
				Enrichments: map[string][]json.RawMessage{
					"message/vnd.clair.map.vulnerability; enricher=clair.cvss": {
						json.RawMessage(`{"CVE-2023-1234": [{"cve": "CVE-2023-1234", "cvss": [{"version": "3.1", "score": 7.5, "vector": "CVSS:3.1/AV:N/AC:L"}]}]}`),
					},
					"message/vnd.clair.map.enrichment; enricher=clair.epss": {
						json.RawMessage(`{"CVE-2023-1234": [{"cve": "CVE-2023-1234", "epss": {"model_version": "v2023.03.01", "date": "2023-05-15", "probability": 0.001, "percentile": 0.45}}]}`),
					},
				},
			},
			pipeline: NewPipeline(WithCSAFEnricher(csaf.NewEnricher(csaf.WithStaticAdvisories(map[string]*csaf.Advisory{
				"RHSA-2023:1234": {Name: "RHSA-2023:1234", Description: "test"},
			})))),
			validate: func(t *testing.T, result *EnrichmentResult, err error) {
				require.NoError(t, err)
				assert.NotEmpty(t, result.NVDVulns)
				assert.NotEmpty(t, result.EPSSItems)
				assert.NotEmpty(t, result.PkgFixedBy)
				assert.NotEmpty(t, result.CSAFAdvisories)
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := t.Context()
			result, err := tc.pipeline.Enrich(ctx, tc.vr)
			tc.validate(t, result, err)
		})
	}
}

func TestPipeline_Enrich_Empty(t *testing.T) {
	tests := map[string]struct {
		vr       *clairclient.VulnerabilityReport
		pipeline *Pipeline
	}{
		"empty report": {
			vr:       &clairclient.VulnerabilityReport{},
			pipeline: NewPipeline(),
		},
		"no enrichments": {
			vr: &clairclient.VulnerabilityReport{
				Packages: map[string]clairclient.Package{
					"pkg1": {ID: "pkg1"},
				},
			},
			pipeline: NewPipeline(),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := t.Context()
			result, err := tc.pipeline.Enrich(ctx, tc.vr)
			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Empty(t, result.NVDVulns)
			assert.Empty(t, result.EPSSItems)
			assert.Empty(t, result.PkgFixedBy)
			assert.Empty(t, result.CSAFAdvisories)
		})
	}
}

func TestPipeline_WithCSAFEnricher(t *testing.T) {
	releaseDate := time.Date(2023, 5, 15, 0, 0, 0, 0, time.UTC)

	csafEnricher := csaf.NewEnricher(csaf.WithStaticAdvisories(map[string]*csaf.Advisory{
		"RHSA-2023:1234": {
			Name:        "RHSA-2023:1234",
			Description: "security update",
			ReleaseDate: releaseDate,
			Severity:    "Important",
		},
	}))

	pipeline := NewPipeline(WithCSAFEnricher(csafEnricher))

	vr := &clairclient.VulnerabilityReport{
		Vulnerabilities: map[string]clairclient.Vulnerability{
			"vuln1": {ID: "vuln1", Name: "RHSA-2023:1234: test"},
		},
	}

	ctx := t.Context()
	result, err := pipeline.Enrich(ctx, vr)
	require.NoError(t, err)
	require.Len(t, result.CSAFAdvisories, 1)
	assert.Equal(t, "security update", result.CSAFAdvisories["vuln1"].Description)
}
