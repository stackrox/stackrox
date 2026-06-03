package csaf

import (
	"testing"
	"time"

	"github.com/stackrox/rox/clair-adapter/clairclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnrich(t *testing.T) {
	releaseDate := time.Date(2023, 5, 15, 0, 0, 0, 0, time.UTC)

	advisory := &Advisory{
		Name:        "RHSA-2023:1234",
		Description: "bash security update",
		ReleaseDate: releaseDate,
		Severity:    "Important",
		CVSSv3: CVSSScore{
			BaseScore: 7.5,
			Vector:    "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H",
		},
		CVSSv2: CVSSScore{
			BaseScore: 5.0,
			Vector:    "AV:N/AC:L/Au:N/C:N/I:N/A:P",
		},
	}

	tests := map[string]struct {
		vr       *clairclient.VulnerabilityReport
		enricher *Enricher
		expected map[string]*Advisory
	}{
		"vuln with RHSA in name matches advisory": {
			vr: &clairclient.VulnerabilityReport{
				Vulnerabilities: map[string]clairclient.Vulnerability{
					"vuln1": {
						ID:   "vuln1",
						Name: "RHSA-2023:1234: bash update",
					},
				},
			},
			enricher: NewEnricher(WithStaticAdvisories(map[string]*Advisory{
				"RHSA-2023:1234": advisory,
			})),
			expected: map[string]*Advisory{
				"vuln1": advisory,
			},
		},
		"RHBA advisory matches": {
			vr: &clairclient.VulnerabilityReport{
				Vulnerabilities: map[string]clairclient.Vulnerability{
					"vuln1": {
						ID:   "vuln1",
						Name: "RHBA-2023:5678: bug fix update",
					},
				},
			},
			enricher: NewEnricher(WithStaticAdvisories(map[string]*Advisory{
				"RHBA-2023:5678": {Name: "RHBA-2023:5678", Description: "Bug fix"},
			})),
			expected: map[string]*Advisory{
				"vuln1": {Name: "RHBA-2023:5678", Description: "Bug fix"},
			},
		},
		"RHEA advisory matches": {
			vr: &clairclient.VulnerabilityReport{
				Vulnerabilities: map[string]clairclient.Vulnerability{
					"vuln1": {
						ID:   "vuln1",
						Name: "RHEA-2023:9999: enhancement advisory",
					},
				},
			},
			enricher: NewEnricher(WithStaticAdvisories(map[string]*Advisory{
				"RHEA-2023:9999": {Name: "RHEA-2023:9999", Description: "Enhancement"},
			})),
			expected: map[string]*Advisory{
				"vuln1": {Name: "RHEA-2023:9999", Description: "Enhancement"},
			},
		},
		"multiple vulns with advisories": {
			vr: &clairclient.VulnerabilityReport{
				Vulnerabilities: map[string]clairclient.Vulnerability{
					"vuln1": {ID: "vuln1", Name: "RHSA-2023:1111: update 1"},
					"vuln2": {ID: "vuln2", Name: "RHSA-2023:2222: update 2"},
				},
			},
			enricher: NewEnricher(WithStaticAdvisories(map[string]*Advisory{
				"RHSA-2023:1111": {Name: "RHSA-2023:1111"},
				"RHSA-2023:2222": {Name: "RHSA-2023:2222"},
			})),
			expected: map[string]*Advisory{
				"vuln1": {Name: "RHSA-2023:1111"},
				"vuln2": {Name: "RHSA-2023:2222"},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := tc.enricher.Enrich(tc.vr)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestEnrich_NoMatch(t *testing.T) {
	tests := map[string]struct {
		vr       *clairclient.VulnerabilityReport
		enricher *Enricher
		expected map[string]*Advisory
	}{
		"CVE-only vuln name with no advisory": {
			vr: &clairclient.VulnerabilityReport{
				Vulnerabilities: map[string]clairclient.Vulnerability{
					"vuln1": {
						ID:   "vuln1",
						Name: "CVE-2023-1234",
					},
				},
			},
			enricher: NewEnricher(WithStaticAdvisories(map[string]*Advisory{
				"RHSA-2023:1234": {Name: "RHSA-2023:1234"},
			})),
			expected: map[string]*Advisory{},
		},
		"RHSA in name but not in advisories": {
			vr: &clairclient.VulnerabilityReport{
				Vulnerabilities: map[string]clairclient.Vulnerability{
					"vuln1": {
						ID:   "vuln1",
						Name: "RHSA-2023:9999: unknown advisory",
					},
				},
			},
			enricher: NewEnricher(WithStaticAdvisories(map[string]*Advisory{
				"RHSA-2023:1234": {Name: "RHSA-2023:1234"},
			})),
			expected: map[string]*Advisory{},
		},
		"empty advisories map": {
			vr: &clairclient.VulnerabilityReport{
				Vulnerabilities: map[string]clairclient.Vulnerability{
					"vuln1": {ID: "vuln1", Name: "RHSA-2023:1234: test"},
				},
			},
			enricher: NewEnricher(),
			expected: map[string]*Advisory{},
		},
		"empty vulnerability report": {
			vr:       &clairclient.VulnerabilityReport{},
			enricher: NewEnricher(),
			expected: map[string]*Advisory{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := tc.enricher.Enrich(tc.vr)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSetAdvisories(t *testing.T) {
	enricher := NewEnricher()

	// Initially empty
	vr := &clairclient.VulnerabilityReport{
		Vulnerabilities: map[string]clairclient.Vulnerability{
			"vuln1": {ID: "vuln1", Name: "RHSA-2023:1234: test"},
		},
	}

	result, err := enricher.Enrich(vr)
	require.NoError(t, err)
	assert.Empty(t, result)

	// Set advisories
	advisories := map[string]*Advisory{
		"RHSA-2023:1234": {Name: "RHSA-2023:1234", Description: "Test advisory"},
	}
	enricher.SetAdvisories(advisories)

	// Now should find advisory
	result, err = enricher.Enrich(vr)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "Test advisory", result["vuln1"].Description)

	// Update advisories
	enricher.SetAdvisories(map[string]*Advisory{})
	result, err = enricher.Enrich(vr)
	require.NoError(t, err)
	assert.Empty(t, result)
}
