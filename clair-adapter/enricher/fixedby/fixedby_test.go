package fixedby

import (
	"testing"

	"github.com/stackrox/rox/clair-adapter/clairclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnrich(t *testing.T) {
	tests := map[string]struct {
		vr       *clairclient.VulnerabilityReport
		expected map[string]string
	}{
		"two vulns with different fixed versions for one package": {
			vr: &clairclient.VulnerabilityReport{
				Packages: map[string]clairclient.Package{
					"pkg1": {ID: "pkg1", Name: "bash", Version: "5.0"},
				},
				Vulnerabilities: map[string]clairclient.Vulnerability{
					"vuln1": {ID: "vuln1", FixedInVersion: "5.2"},
					"vuln2": {ID: "vuln2", FixedInVersion: "5.3"},
				},
				PackageVulnerabilities: map[string][]string{
					"pkg1": {"vuln1", "vuln2"},
				},
			},
			expected: map[string]string{
				"pkg1": "5.3",
			},
		},
		"multiple packages with different fixed versions": {
			vr: &clairclient.VulnerabilityReport{
				Packages: map[string]clairclient.Package{
					"pkg1": {ID: "pkg1", Name: "bash"},
					"pkg2": {ID: "pkg2", Name: "curl"},
				},
				Vulnerabilities: map[string]clairclient.Vulnerability{
					"vuln1": {ID: "vuln1", FixedInVersion: "5.2"},
					"vuln2": {ID: "vuln2", FixedInVersion: "5.3"},
					"vuln3": {ID: "vuln3", FixedInVersion: "7.1"},
				},
				PackageVulnerabilities: map[string][]string{
					"pkg1": {"vuln1", "vuln2"},
					"pkg2": {"vuln3"},
				},
			},
			expected: map[string]string{
				"pkg1": "5.3",
				"pkg2": "7.1",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := Enrich(tc.vr)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestEnrich_NoFixedVersion(t *testing.T) {
	tests := map[string]struct {
		vr       *clairclient.VulnerabilityReport
		expected map[string]string
	}{
		"vuln with empty fixed version": {
			vr: &clairclient.VulnerabilityReport{
				Packages: map[string]clairclient.Package{
					"pkg1": {ID: "pkg1", Name: "bash"},
				},
				Vulnerabilities: map[string]clairclient.Vulnerability{
					"vuln1": {ID: "vuln1", FixedInVersion: ""},
				},
				PackageVulnerabilities: map[string][]string{
					"pkg1": {"vuln1"},
				},
			},
			expected: map[string]string{},
		},
		"all vulns without fixed version": {
			vr: &clairclient.VulnerabilityReport{
				Packages: map[string]clairclient.Package{
					"pkg1": {ID: "pkg1"},
					"pkg2": {ID: "pkg2"},
				},
				Vulnerabilities: map[string]clairclient.Vulnerability{
					"vuln1": {ID: "vuln1", FixedInVersion: ""},
					"vuln2": {ID: "vuln2", FixedInVersion: ""},
				},
				PackageVulnerabilities: map[string][]string{
					"pkg1": {"vuln1"},
					"pkg2": {"vuln2"},
				},
			},
			expected: map[string]string{},
		},
		"mixed fixed and unfixed vulns": {
			vr: &clairclient.VulnerabilityReport{
				Packages: map[string]clairclient.Package{
					"pkg1": {ID: "pkg1"},
				},
				Vulnerabilities: map[string]clairclient.Vulnerability{
					"vuln1": {ID: "vuln1", FixedInVersion: ""},
					"vuln2": {ID: "vuln2", FixedInVersion: "5.3"},
				},
				PackageVulnerabilities: map[string][]string{
					"pkg1": {"vuln1", "vuln2"},
				},
			},
			expected: map[string]string{
				"pkg1": "5.3",
			},
		},
		"empty report": {
			vr:       &clairclient.VulnerabilityReport{},
			expected: map[string]string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := Enrich(tc.vr)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}
