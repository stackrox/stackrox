package scannerv4

import (
	"testing"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/scannerv4/rhcos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRHCOSComponentConversion(t *testing.T) {
	for name, tc := range map[string]struct {
		canonical, legacy bool
		canonicalCVEs     []string
		legacyCVEs        []string
		want              map[string]string
	}{
		"canonical wins per CVE":         {true, true, []string{"shared", "canonical-only"}, []string{"shared", "legacy-only"}, map[string]string{"shared": "canonical", "canonical-only": "canonical", "legacy-only": "legacy"}},
		"empty canonical retains legacy": {true, true, nil, []string{"legacy-only"}, map[string]string{"legacy-only": "legacy"}},
		"empty legacy":                   {true, true, []string{"canonical-only"}, nil, map[string]string{"canonical-only": "canonical"}},
		"migration complete":             {true, false, []string{"canonical-only"}, nil, map[string]string{"canonical-only": "canonical"}},
		"no findings":                    {true, true, nil, nil, map[string]string{}},
	} {
		t.Run(name, func(t *testing.T) {
			r := &v4.VulnerabilityReport{
				Contents:               &v4.Contents{Packages: map[string]*v4.Package{}},
				Vulnerabilities:        map[string]*v4.VulnerabilityReport_Vulnerability{},
				PackageVulnerabilities: map[string]*v4.StringList{},
			}
			addPair := func(id, srcID, name, origin string, cves []string) {
				r.Contents.Packages[id] = &v4.Package{Id: id, Name: name, Version: "9.6.20260324-0"}
				r.Contents.Packages[srcID] = &v4.Package{Id: srcID, Name: name, Version: "9.6.20260324-0"}
				ids := &v4.StringList{}
				for _, cve := range cves {
					vulnID := origin + "-" + cve
					v := &v4.VulnerabilityReport_Vulnerability{Name: cve, Description: origin}
					// In particular, an unfixed canonical assertion must not acquire
					// a fixed version from the legacy assertion for the same CVE.
					if origin == "legacy" {
						v.FixedInVersion = "legacy-fix"
					}
					r.Vulnerabilities[vulnID] = v
					ids.Values = append(ids.Values, vulnID)
				}
				r.PackageVulnerabilities[id] = ids
				// A source entry must neither create another image component nor
				// duplicate findings already associated with its binary entry.
				r.PackageVulnerabilities[srcID] = ids
			}
			if tc.canonical {
				addPair(rhcos.PackageID, rhcos.SourceID, "openshift/ose-rhel-coreos-9", "canonical", tc.canonicalCVEs)
			}
			if tc.legacy {
				addPair(rhcos.LegacyPackageID, rhcos.LegacySourceID, "rhcos", "legacy", tc.legacyCVEs)
			}
			// Ordinary packages retain their own copy of an overlapping CVE.
			r.Contents.Packages["rpm"] = &v4.Package{Id: "rpm", Name: "kernel", Version: "rpm-version"}
			r.Vulnerabilities["rpm-vuln"] = &v4.VulnerabilityReport_Vulnerability{Name: "shared", Description: "rpm"}
			r.PackageVulnerabilities["rpm"] = &v4.StringList{Values: []string{"rpm-vuln"}}
			before := r.CloneVT()
			for _, deprecated := range []bool{false, true} {
				if deprecated {
					for _, pkg := range r.GetContents().GetPackages() {
						r.Contents.PackagesDEPRECATED = append(r.GetContents().GetPackagesDEPRECATED(), pkg)
					}
					r.Contents.Packages = nil
					before = r.CloneVT()
				}
				components := toStorageComponents(r)
				require.Len(t, components, 2)
				byName := map[string]*storage.EmbeddedNodeScanComponent{}
				for _, component := range components {
					byName[component.GetName()] = component
				}
				image := byName["openshift/ose-rhel-coreos-9"]
				require.NotNil(t, image)
				assert.Equal(t, "9.6.20260324-0", image.GetVersion())
				require.Len(t, image.GetVulns(), len(tc.want))
				for _, vuln := range image.GetVulns() {
					origin, ok := tc.want[vuln.GetCve()]
					require.True(t, ok)
					assert.Equal(t, origin, vuln.GetSummary())
					if origin == "canonical" {
						assert.Empty(t, vuln.GetFixedBy())
					} else {
						assert.Equal(t, "legacy-fix", vuln.GetFixedBy())
					}
				}
				require.Len(t, byName["kernel"].GetVulns(), 1)
				assert.Equal(t, "rpm", byName["kernel"].GetVulns()[0].GetSummary())
				protoassert.Equal(t, before, r)
			}
		})
	}
}

func TestRHCOSLegacyOnlyConversion(t *testing.T) {
	// Reports from older Compliance instances retain their existing behavior.
	r := &v4.VulnerabilityReport{Contents: &v4.Contents{Packages: map[string]*v4.Package{
		rhcos.LegacyPackageID: {Id: rhcos.LegacyPackageID, Name: "rhcos", Version: "old"},
	}}}
	components := toStorageComponents(r)
	require.Len(t, components, 1)
	assert.Equal(t, "rhcos", components[0].GetName())
}

func TestRHCOSSourceFindingPrecedence(t *testing.T) {
	r := &v4.VulnerabilityReport{
		Contents: &v4.Contents{Packages: map[string]*v4.Package{
			rhcos.PackageID:       {Id: rhcos.PackageID, Name: "openshift/ose-rhel-coreos-8"},
			rhcos.SourceID:        {Id: rhcos.SourceID, Name: "openshift/ose-rhel-coreos-8"},
			rhcos.LegacyPackageID: {Id: rhcos.LegacyPackageID, Name: "rhcos"},
			rhcos.LegacySourceID:  {Id: rhcos.LegacySourceID, Name: "rhcos"},
		}},
		Vulnerabilities: map[string]*v4.VulnerabilityReport_Vulnerability{
			"canonical": {Name: "CVE-shared", Description: "canonical source", FixedInVersion: "canonical-fix"},
			"legacy":    {Name: "CVE-shared", Description: "legacy binary", FixedInVersion: "legacy-fix"},
			"source":    {Name: "CVE-legacy-source-only", Description: "legacy source"},
		},
		PackageVulnerabilities: map[string]*v4.StringList{
			rhcos.SourceID:        {Values: []string{"canonical"}},
			rhcos.LegacyPackageID: {Values: []string{"legacy"}},
			rhcos.LegacySourceID:  {Values: []string{"source"}},
		},
	}
	components := toStorageComponents(r)
	require.Len(t, components, 1)
	require.Len(t, components[0].GetVulns(), 2)
	assert.Equal(t, "canonical source", components[0].GetVulns()[0].GetSummary())
	assert.Equal(t, "canonical-fix", components[0].GetVulns()[0].GetFixedBy())
	assert.Equal(t, "CVE-legacy-source-only", components[0].GetVulns()[1].GetCve())
}
