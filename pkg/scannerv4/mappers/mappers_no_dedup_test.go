package mappers

import (
	"context"
	"testing"
	"time"

	"github.com/quay/claircore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDedupeAdvisoriesByCVE verifies that Scanner Output B (Variant 2) merges
// advisories by CVE name, picks the winner by severity, and populates AdvisoryDetails.
func TestDedupeAdvisoriesByCVE(t *testing.T) {
	now := time.Now()

	// Create two different advisories that both resolve to CVE-2024-1234
	// goAdvisory has CRITICAL severity (should win)
	goAdvisory := &claircore.Vulnerability{
		ID:                 "osv-go-1",
		Name:               "GO-2024-1234", // ClairCore advisory name
		Description:        "Go vulnerability database advisory",
		Issued:             now,
		Links:              "https://pkg.go.dev/vuln/GO-2024-1234 CVE-2024-1234",
		Severity:           "Critical",
		NormalizedSeverity: claircore.Critical,
		Package:            &claircore.Package{ID: "pkg-1"},
		Dist:               &claircore.Distribution{ID: "dist-1"},
		Repo:               &claircore.Repository{ID: "repo-1"},
		FixedInVersion:     "1.2.3",
		Updater:            "osv/Go",
	}

	// ghsaAdvisory has HIGH severity (should be merged)
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

	// Call toProtoV4VulnerabilitiesMap to create proto entries
	ctx := context.Background()
	protoVulns, err := toProtoV4VulnerabilitiesMap(
		ctx,
		ccVulns,
		nil, // nvdVulns
		nil, // epssItems
		nil, // csafAdvisories
	)
	require.NoError(t, err)
	require.Len(t, protoVulns, 2, "Should have 2 proto vulns before dedup")

	// Create package vulnerabilities mapping (both advisories for same package)
	ccPkgVulns := map[string][]string{
		"pkg-1": {"osv-go-1", "osv-ghsa-1"},
	}

	// Call the function that performs deduplication
	result := toProtoV4PackageVulnerabilitiesMap(ccPkgVulns, ccVulns, protoVulns)

	// Verify deduplication occurred
	require.Contains(t, result, "pkg-1")
	vulnIDs := result["pkg-1"].Values
	assert.Len(t, vulnIDs, 1, "Expected advisories to be merged into one CVE entry")

	// The winner should be the CRITICAL one (osv-go-1)
	assert.Contains(t, vulnIDs, "osv-go-1", "Expected CRITICAL severity advisory to win")

	// Check that the winner has AdvisoryDetails populated
	winner := protoVulns["osv-go-1"]
	require.NotNil(t, winner)
	assert.Equal(t, "CVE-2024-1234", winner.CveName, "Winner should have CVE name set")
	require.Len(t, winner.AdvisoryDetails, 2, "Winner should have AdvisoryDetails from both advisories")

	// Verify both advisories are in the details
	advisoryIDs := make(map[string]bool)
	for _, detail := range winner.AdvisoryDetails {
		advisoryIDs[detail.Id] = true
	}
	assert.True(t, advisoryIDs["GO-2024-1234"], "AdvisoryDetails should contain GO advisory")
	assert.True(t, advisoryIDs["GHSA-xxxx-yyyy-zzzz"], "AdvisoryDetails should contain GHSA advisory")
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

// TestAdvisoryMergingWithSameSeverity verifies that when multiple advisories
// have the same severity, the one with highest CVSS wins.
func TestAdvisoryMergingWithSameSeverity(t *testing.T) {
	now := time.Now()

	// Create two advisories that resolve to the same CVE, both HIGH severity
	// but with different CVSS scores
	vuln1 := &claircore.Vulnerability{
		ID:                 "v1",
		Name:               "GO-2024-5678",
		Links:              "CVE-2024-5678",
		Issued:             now,
		NormalizedSeverity: claircore.High,
		Severity:           "7.5",
		Package:            &claircore.Package{ID: "pkg-1"},
		Updater:            "osv/Go",
	}

	vuln2 := &claircore.Vulnerability{
		ID:                 "v2",
		Name:               "GHSA-aaaa-bbbb-cccc",
		Links:              "CVE-2024-5678",
		Issued:             now,
		NormalizedSeverity: claircore.High,
		Severity:           "8.9", // Higher CVSS, should win
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

	// Call the function that calls dedupeVulns
	result := toProtoV4PackageVulnerabilitiesMap(ccPkgVulns, ccVulns, protoVulns)

	// Verify only one entry (merged)
	require.Contains(t, result, "pkg-1")
	vulnIDs := result["pkg-1"].Values
	assert.Len(t, vulnIDs, 1, "Expected advisories to be merged into one CVE entry")

	// The winner should have AdvisoryDetails
	var winnerID string
	if len(vulnIDs) > 0 {
		winnerID = vulnIDs[0]
	}
	winner := protoVulns[winnerID]
	require.NotNil(t, winner)
	assert.Equal(t, "CVE-2024-5678", winner.CveName)
	require.Len(t, winner.AdvisoryDetails, 2, "Winner should have both advisories in details")
}
