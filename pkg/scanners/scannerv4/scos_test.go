package scannerv4

import (
	"testing"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scannerv4/repositorytocpe"
	"github.com/stackrox/rox/pkg/scannerv4/scos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSCOSNodeInventoryScan(t *testing.T) {
	index := &v4.IndexReport{Contents: &v4.Contents{
		Packages: map[string]*v4.Package{"1": {Id: "1", Name: "openssl", Version: "1.2.3"}},
		// Older collectors may only send the deprecated environment fields.
		EnvironmentsDEPRECATED: map[string]*v4.Environment_List{"1": {Environments: []*v4.Environment{{PackageDb: "sqlite:usr/share/rpm"}}}},
	}}
	mapping := &repositorytocpe.MappingFile{Data: map[string]repositorytocpe.Repo{
		"rhel-10-for-x86_64-baseos-rpms":    {CPEs: []string{"cpe:/o:redhat:enterprise_linux:10::baseos"}},
		"rhel-10-for-x86_64-appstream-rpms": {CPEs: []string{"cpe:/a:redhat:enterprise_linux:10::appstream"}},
	}}
	// The same package/version in RHCOS and SCOS must get different scan scopes.
	prepared, err := scos.AddCompatibility(index, scos.Profile{Major: "10", Arch: "x86_64"}, mapping)
	require.NoError(t, err)
	contents := prepared.GetContents()
	assert.Empty(t, index.GetContents().GetRepositories())
	assert.Empty(t, index.GetContents().GetEnvironmentsDEPRECATED()["1"].GetEnvironments()[0].GetRepositoryIds())
	assert.Equal(t, "10", contents.GetDistributions()[scos.DistributionID].GetVersionId())
	assert.Len(t, contents.GetRepositories(), 2)
	assert.Len(t, contents.GetEnvironments()["1"].GetEnvironments()[0].GetRepositoryIds(), 2)
	assert.Equal(t, contents.GetEnvironments(), contents.GetEnvironmentsDEPRECATED())
	vr := &v4.VulnerabilityReport{
		Contents:               contents,
		PackageVulnerabilities: map[string]*v4.StringList{"1": {Values: []string{"v"}}},
		Vulnerabilities:        map[string]*v4.VulnerabilityReport_Vulnerability{"v": {Name: "CVE-2025-1234", FixedInVersion: "3.0.1"}},
	}
	vr.Notes = []v4.VulnerabilityReport_Note{v4.VulnerabilityReport_NOTE_OS_UNSUPPORTED}
	scan := toNodeScan(vr, scos.Name+" 10.0.20260414-0 (Coughlan)")
	assert.Equal(t, "scos:10:x86_64", scan.GetOperatingSystem())
	assert.Contains(t, scan.GetNotes(), storage.NodeScan_UNSUPPORTED)
	require.Len(t, scan.GetComponents(), 1)
	require.Len(t, scan.GetComponents()[0].GetVulns(), 1)
	assert.Equal(t, "CVE-2025-1234", scan.GetComponents()[0].GetVulns()[0].GetCve())
	assert.Empty(t, scan.GetComponents()[0].GetVulns()[0].GetFixedBy())
	rhcos := toNodeScan(vr, "Red Hat Enterprise Linux CoreOS 419.96.20250915-0")
	assert.Equal(t, "rhcos:4.19", rhcos.GetOperatingSystem())
	assert.Equal(t, "3.0.1", rhcos.GetComponents()[0].GetVulns()[0].GetFixedBy())
}
