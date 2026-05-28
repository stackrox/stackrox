package mappers

import (
	"context"
	"testing"
	"time"

	"github.com/quay/claircore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNoDedupeAdvisories verifies that Scanner Output A preserves all advisories
// (no CVE dedup) and populates CveName, AdvisoryId, and SourceName fields.
func TestNoDedupeAdvisories(t *testing.T) {
	now := time.Now()

	// Create two different advisories that both resolve to CVE-2024-1234
	goAdvisory := &claircore.Vulnerability{
		ID:                 "osv-go-1",
		Name:               "GO-2024-1234", // ClairCore advisory name
		Description:        "Go vulnerability database advisory",
		Issued:             now,
		Links:              "https://pkg.go.dev/vuln/GO-2024-1234 CVE-2024-1234",
		Severity:           "High",
		NormalizedSeverity: claircore.High,
		Package:            &claircore.Package{ID: "pkg-1"},
		Dist:               &claircore.Distribution{ID: "dist-1"},
		Repo:               &claircore.Repository{ID: "repo-1"},
		FixedInVersion:     "1.2.3",
		Updater:            "osv/Go",
	}

	ghsaAdvisory := &claircore.Vulnerability{
		ID:                 "osv-ghsa-1",
		Name:               "GHSA-xxxx-yyyy-zzzz", // Different advisory name
		Description:        "GitHub Security Advisory",
		Issued:             now,
		Links:              "https://github.com/advisories/GHSA-xxxx-yyyy-zzzz CVE-2024-1234",
		Severity:           "High",
		NormalizedSeverity: claircore.High,
		Package:            &claircore.Package{ID: "pkg-1"},
		Dist:               &claircore.Distribution{ID: "dist-1"},
		Repo:               &claircore.Repository{ID: "repo-1"},
		FixedInVersion:     "1.2.3",
		Updater:            "osv/Go",
	}

	ccVulns := map[string]*claircore.Vulnerability{
		"osv-go-1":   goAdvisory,
		"osv-ghsa-1": ghsaAdvisory,
	}

	// Call toProtoV4VulnerabilitiesMap which should create both entries
	ctx := context.Background()
	result, err := toProtoV4VulnerabilitiesMap(
		ctx,
		ccVulns,
		nil, // nvdVulns
		nil, // epssItems
		nil, // csafAdvisories
	)
	require.NoError(t, err)

	// Assert both advisories are present in output (no dedup)
	require.Len(t, result, 2, "Expected both advisories to be preserved")
	require.Contains(t, result, "osv-go-1")
	require.Contains(t, result, "osv-ghsa-1")

	// Check the GO-2024-1234 advisory
	goProto := result["osv-go-1"]
	assert.Equal(t, "CVE-2024-1234", goProto.CveName, "CveName should be resolved CVE")
	assert.Equal(t, "GO-2024-1234", goProto.AdvisoryId, "AdvisoryId should be raw ClairCore name")
	assert.Equal(t, "Go Vulnerability DB", goProto.SourceName, "SourceName should be human-readable")

	// Check the GHSA advisory
	ghsaProto := result["osv-ghsa-1"]
	assert.Equal(t, "CVE-2024-1234", ghsaProto.CveName, "CveName should be resolved CVE")
	assert.Equal(t, "GHSA-xxxx-yyyy-zzzz", ghsaProto.AdvisoryId, "AdvisoryId should be raw ClairCore name")
	assert.Equal(t, "Go Vulnerability DB", ghsaProto.SourceName, "SourceName should be human-readable")
}

// TestUpdaterDisplayName verifies the updater name to display name conversion.
func TestUpdaterDisplayName(t *testing.T) {
	tests := map[string]struct {
		updater string
		want    string
	}{
		"Go OSV":       {updater: "osv/Go", want: "Go Vulnerability DB"},
		"PyPI OSV":     {updater: "osv/PyPI", want: "PyPI Advisory DB"},
		"npm OSV":      {updater: "osv/npm", want: "npm Advisory DB"},
		"Maven OSV":    {updater: "osv/Maven", want: "Maven Advisory DB"},
		"RubyGems OSV": {updater: "osv/RubyGems", want: "RubyGems Advisory DB"},
		"NuGet OSV":    {updater: "osv/NuGet", want: "NuGet Advisory DB"},
		"Unknown OSV":  {updater: "osv/Rust", want: "Rust Advisory DB"},
		"Red Hat VEX":  {updater: "rhel-vex", want: "Red Hat VEX"},
		"Debian":       {updater: "debian", want: "Debian Security Tracker"},
		"Ubuntu":       {updater: "ubuntu", want: "Ubuntu Security Tracker"},
		"Alpine":       {updater: "alpine", want: "Alpine SecDB"},
		"AWS":          {updater: "aws", want: "AWS Security Advisory"},
		"Unknown":      {updater: "unknown-updater", want: "unknown-updater"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := updaterDisplayName(tt.updater)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestNoDedupeInPackageVulnerabilities verifies that package vulnerability lists
// preserve all advisories when dedupeVulns is disabled.
func TestNoDedupeInPackageVulnerabilities(t *testing.T) {
	now := time.Now()

	// Create two advisories that resolve to the same CVE
	vuln1 := &claircore.Vulnerability{
		ID:                 "v1",
		Name:               "GO-2024-5678",
		Links:              "CVE-2024-5678",
		Issued:             now,
		NormalizedSeverity: claircore.High,
		Package:            &claircore.Package{ID: "pkg-1"},
		Updater:            "osv/Go",
	}

	vuln2 := &claircore.Vulnerability{
		ID:                 "v2",
		Name:               "GHSA-aaaa-bbbb-cccc",
		Links:              "CVE-2024-5678",
		Issued:             now,
		NormalizedSeverity: claircore.High,
		Package:            &claircore.Package{ID: "pkg-1"},
		Updater:            "osv/Go",
	}

	ccVulns := map[string]*claircore.Vulnerability{
		"v1": vuln1,
		"v2": vuln2,
	}

	// Create proto vulns
	ctx := context.Background()
	protoVulns, err := toProtoV4VulnerabilitiesMap(ctx, ccVulns, nil, nil, nil)
	require.NoError(t, err)

	// Create package vulnerabilities mapping
	ccPkgVulns := map[string][]string{
		"pkg-1": {"v1", "v2"},
	}

	// Call the function that normally calls dedupeVulns
	result := toProtoV4PackageVulnerabilitiesMap(ccPkgVulns, ccVulns, protoVulns)

	// Verify both advisory IDs are present (no dedup)
	require.Contains(t, result, "pkg-1")
	vulnIDs := result["pkg-1"].Values
	assert.Len(t, vulnIDs, 2, "Expected both advisory IDs to be preserved in package vulnerability list")
	assert.Contains(t, vulnIDs, "v1")
	assert.Contains(t, vulnIDs, "v2")
}
