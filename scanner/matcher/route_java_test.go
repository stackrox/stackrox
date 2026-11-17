package matcher

import (
	"testing"

	"github.com/quay/claircore"
	"github.com/quay/claircore/java"
	rheljava "github.com/quay/claircore/rhel/java"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRouteRedHatJavaToOSV(t *testing.T) {
	maven := java.Repository
	maven.ID = "maven"
	ir := &claircore.IndexReport{
		Packages: map[string]*claircore.Package{
			"sbom": {ID: "sbom", Name: "org.example:library", PackageDB: "sbom:app.cdx.json"},
			"jar":  {ID: "jar", Name: "org.example:other", PackageDB: "maven:other.jar"},
			"rpm":  {ID: "rpm", Name: "example", PackageDB: "rpm"},
		},
		Repositories: map[string]*claircore.Repository{
			"product1": {ID: "product1", Key: rheljava.RepositoryKey},
			"product2": {ID: "product2", Key: rheljava.RepositoryKey},
			"maven":    &maven,
		},
		Environments: map[string][]*claircore.Environment{
			"sbom": {{PackageDB: "sbom:app.cdx.json", RepositoryIDs: []string{"product1", "product2"}}},
			"jar":  {{PackageDB: "maven:other.jar", RepositoryIDs: []string{"maven"}}},
			"rpm":  {{PackageDB: "rpm", RepositoryIDs: []string{"product1"}}},
		},
	}

	got := routeRedHatJavaToOSV(ir)
	require.NotSame(t, ir, got)
	assert.Equal(t, []string{"maven"}, got.Environments["sbom"][0].RepositoryIDs)
	assert.Equal(t, []string{"maven"}, got.Environments["jar"][0].RepositoryIDs)
	assert.Equal(t, []string{"product1"}, got.Environments["rpm"][0].RepositoryIDs)
	assert.Equal(t, []string{"product1", "product2"}, ir.Environments["sbom"][0].RepositoryIDs)

	var sbomRecord *claircore.IndexRecord
	for _, record := range got.IndexRecords() {
		if record.Package.ID == "sbom" {
			sbomRecord = record
		}
	}
	require.NotNil(t, sbomRecord)
	assert.True(t, (&java.Matcher{}).Filter(sbomRecord))
	assert.False(t, (&rheljava.RedHatMatcher{}).Filter(sbomRecord))

	got.Environments["sbom"][0].RepositoryIDs[0] = "changed"
	assert.Equal(t, []string{"product1", "product2"}, ir.Environments["sbom"][0].RepositoryIDs)
}

func TestRouteRedHatJavaToOSVCreatesMavenRepository(t *testing.T) {
	ir := &claircore.IndexReport{
		Packages: map[string]*claircore.Package{
			"sbom": {ID: "sbom", Name: "org.example:library", PackageDB: "sbom:app.cdx.json"},
		},
		Repositories: map[string]*claircore.Repository{
			"product":            {ID: "product", Key: rheljava.RepositoryKey},
			"stackrox-osv-maven": {ID: "stackrox-osv-maven", Name: "another"},
		},
		Environments: map[string][]*claircore.Environment{
			"sbom": {{RepositoryIDs: []string{"product"}}},
		},
	}

	got := routeRedHatJavaToOSV(ir)
	require.NotSame(t, ir, got)
	require.Equal(t, []string{"stackrox-osv-maven-1"}, got.Environments["sbom"][0].RepositoryIDs)
	assert.Equal(t, java.Repository.Name, got.Repositories["stackrox-osv-maven-1"].Name)
	assert.Equal(t, java.Repository.URI, got.Repositories["stackrox-osv-maven-1"].URI)
	assert.NotContains(t, ir.Repositories, "stackrox-osv-maven-1")
	assert.Equal(t, []string{"product"}, ir.Environments["sbom"][0].RepositoryIDs)
}

func TestRouteRedHatJavaToOSVLeavesOtherReportsUntouched(t *testing.T) {
	ir := &claircore.IndexReport{
		Packages: map[string]*claircore.Package{
			"sbom": {ID: "sbom", PackageDB: "sbom:app.cdx.json"},
		},
		Environments: map[string][]*claircore.Environment{
			"sbom": {{RepositoryIDs: nil}},
		},
	}
	assert.Same(t, ir, routeRedHatJavaToOSV(ir))
}
