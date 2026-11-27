package scannerv4

import (
	"testing"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stretchr/testify/assert"
)

func TestFixedContainerAssessmentScope(t *testing.T) {
	alias := []*v4.VulnerabilityReport_Alias{{Space: "cve", Name: "2025-68121"}}
	r := &v4.VulnerabilityReport{
		Contents: &v4.Contents{
			Packages: map[string]*v4.Package{
				"image": {Kind: "ancestry"}, "go": {Kind: "binary"}, "rpm": {Kind: "binary"}, "other": {Kind: "binary"},
			},
			Environments: map[string]*v4.Environment_List{},
		},
		Vulnerabilities: map[string]*v4.VulnerabilityReport_Vulnerability{
			"fixed": {Updater: "rhel-vex", Aliases: alias, FixedInVersion: "1775604309"},
			"go":    {Updater: "osv/go", Aliases: alias},
			"rpm":   {Updater: "rhel-vex", Aliases: alias},
			"other": {Updater: "osv/go", Aliases: []*v4.VulnerabilityReport_Alias{{Space: "cve", Name: "unrelated-control"}}},
		},
		PackageVulnerabilities: map[string]*v4.StringList{
			"go": {Values: []string{"go"}}, "rpm": {Values: []string{"rpm"}}, "other": {Values: []string{"other"}},
		},
		PackageNotVulnerable: map[string]*v4.StringList{"image": {Values: []string{"fixed"}}},
	}
	for id := range r.Contents.Packages {
		r.Contents.Environments[id] = &v4.Environment_List{Environments: []*v4.Environment{{IntroducedIn: "layer"}}}
	}
	filterNotAffectedVulnerabilities(r, map[string]int32{"layer": 0})
	assert.Empty(t, r.PackageVulnerabilities["go"].GetValues(), "covered upstream finding remains")
	assert.Equal(t, []string{"rpm"}, r.PackageVulnerabilities["rpm"].GetValues(), "fixed container assessment masked an independent RPM finding")
	assert.Equal(t, []string{"other"}, r.PackageVulnerabilities["other"].GetValues(), "fixed container assessment masked an unrelated CVE")
}
