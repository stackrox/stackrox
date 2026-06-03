package mappers

import (
	"testing"

	"github.com/stackrox/rox/clair-adapter/clairclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToProtoIndexReport(t *testing.T) {
	clairReport := &clairclient.IndexReport{
		ManifestHash: "sha256:abc123",
		State:        "IndexFinished",
		Success:      true,
		Err:          "",
		Packages: map[string]clairclient.Package{
			"pkg1": {
				ID:      "pkg1",
				Name:    "curl",
				Version: "7.68.0-1ubuntu2.4",
				Kind:    "binary",
				Arch:    "amd64",
				Source: &clairclient.Package{
					ID:      "src1",
					Name:    "curl",
					Version: "7.68.0-1ubuntu2.4",
					Kind:    "source",
				},
				Module:         "",
				CPE:            "cpe:2.3:a:haxx:curl:7.68.0:*:*:*:*:*:*:*",
				PackageDB:      "var/lib/dpkg/status",
				RepositoryHint: "ubuntu",
				NormalizedVersion: clairclient.NormalizedVersion{
					Kind: "deb",
					V:    []int{7, 68, 0, 1},
				},
			},
		},
		Distributions: map[string]clairclient.Distribution{
			"dist1": {
				ID:              "dist1",
				DID:             "ubuntu",
				Name:            "Ubuntu",
				Version:         "20.04",
				VersionCodeName: "focal",
				VersionID:       "20.04",
				Arch:            "amd64",
				CPE:             "cpe:/o:canonical:ubuntu_linux:20.04",
				PrettyName:      "Ubuntu 20.04 LTS",
			},
		},
		Repositories: map[string]clairclient.Repository{
			"repo1": {
				ID:   "repo1",
				Name: "Ubuntu 20.04",
				Key:  "ubuntu-key",
				URI:  "http://archive.ubuntu.com/ubuntu",
				CPE:  "cpe:/o:canonical:ubuntu_linux:20.04",
			},
		},
		Environments: map[string][]clairclient.Environment{
			"pkg1": {
				{
					PackageDB:      "var/lib/dpkg/status",
					IntroducedIn:   "sha256:layer1",
					DistributionID: "dist1",
					RepositoryIDs:  []string{"repo1"},
				},
			},
		},
	}

	protoReport, err := ToProtoIndexReport(clairReport)
	require.NoError(t, err)
	require.NotNil(t, protoReport)

	// Verify top-level fields
	assert.Equal(t, "sha256:abc123", protoReport.HashId)
	assert.Equal(t, "IndexFinished", protoReport.State)
	assert.True(t, protoReport.Success)
	assert.Empty(t, protoReport.Err)

	// Verify contents exist
	require.NotNil(t, protoReport.Contents)

	// Verify package mapping
	require.Len(t, protoReport.Contents.Packages, 1)
	pkg, ok := protoReport.Contents.Packages["pkg1"]
	require.True(t, ok)
	assert.Equal(t, "pkg1", pkg.Id)
	assert.Equal(t, "curl", pkg.Name)
	assert.Equal(t, "7.68.0-1ubuntu2.4", pkg.Version)
	assert.Equal(t, "binary", pkg.Kind)
	assert.Equal(t, "amd64", pkg.Arch)
	assert.Equal(t, "cpe:2.3:a:haxx:curl:7.68.0:*:*:*:*:*:*:*", pkg.Cpe)
	assert.Equal(t, "var/lib/dpkg/status", pkg.PackageDb)
	assert.Equal(t, "ubuntu", pkg.RepositoryHint)
	assert.Empty(t, pkg.FixedInVersion) // Should be empty for index reports

	// Verify source package
	require.NotNil(t, pkg.Source)
	assert.Equal(t, "src1", pkg.Source.Id)
	assert.Equal(t, "curl", pkg.Source.Name)
	assert.Equal(t, "source", pkg.Source.Kind)

	// Verify normalized version (int -> int32 conversion)
	require.NotNil(t, pkg.NormalizedVersion)
	assert.Equal(t, "deb", pkg.NormalizedVersion.Kind)
	assert.Equal(t, []int32{7, 68, 0, 1}, pkg.NormalizedVersion.V)

	// Verify distribution mapping
	require.Len(t, protoReport.Contents.Distributions, 1)
	dist, ok := protoReport.Contents.Distributions["dist1"]
	require.True(t, ok)
	assert.Equal(t, "dist1", dist.Id)
	assert.Equal(t, "ubuntu", dist.Did)
	assert.Equal(t, "Ubuntu", dist.Name)
	assert.Equal(t, "20.04", dist.Version)
	assert.Equal(t, "focal", dist.VersionCodeName)
	assert.Equal(t, "20.04", dist.VersionId)
	assert.Equal(t, "amd64", dist.Arch)
	assert.Equal(t, "cpe:/o:canonical:ubuntu_linux:20.04", dist.Cpe)
	assert.Equal(t, "Ubuntu 20.04 LTS", dist.PrettyName)

	// Verify repository mapping
	require.Len(t, protoReport.Contents.Repositories, 1)
	repo, ok := protoReport.Contents.Repositories["repo1"]
	require.True(t, ok)
	assert.Equal(t, "repo1", repo.Id)
	assert.Equal(t, "Ubuntu 20.04", repo.Name)
	assert.Equal(t, "ubuntu-key", repo.Key)
	assert.Equal(t, "http://archive.ubuntu.com/ubuntu", repo.Uri)
	assert.Equal(t, "cpe:/o:canonical:ubuntu_linux:20.04", repo.Cpe)

	// Verify environment mapping
	require.Len(t, protoReport.Contents.Environments, 1)
	envList, ok := protoReport.Contents.Environments["pkg1"]
	require.True(t, ok)
	require.NotNil(t, envList)
	require.Len(t, envList.Environments, 1)
	env := envList.Environments[0]
	assert.Equal(t, "var/lib/dpkg/status", env.PackageDb)
	assert.Equal(t, "sha256:layer1", env.IntroducedIn)
	assert.Equal(t, "dist1", env.DistributionId)
	assert.Equal(t, []string{"repo1"}, env.RepositoryIds)
}

func TestToProtoIndexReport_EmptyReport(t *testing.T) {
	clairReport := &clairclient.IndexReport{
		ManifestHash:  "sha256:empty",
		State:         "IndexFinished",
		Success:       true,
		Packages:      map[string]clairclient.Package{},
		Distributions: map[string]clairclient.Distribution{},
		Repositories:  map[string]clairclient.Repository{},
		Environments:  map[string][]clairclient.Environment{},
	}

	protoReport, err := ToProtoIndexReport(clairReport)
	require.NoError(t, err)
	require.NotNil(t, protoReport)

	assert.Equal(t, "sha256:empty", protoReport.HashId)
	assert.Equal(t, "IndexFinished", protoReport.State)
	assert.True(t, protoReport.Success)

	// Verify empty contents don't cause panics
	require.NotNil(t, protoReport.Contents)
	assert.Empty(t, protoReport.Contents.Packages)
	assert.Empty(t, protoReport.Contents.Distributions)
	assert.Empty(t, protoReport.Contents.Repositories)
	assert.Empty(t, protoReport.Contents.Environments)
}

func TestToProtoIndexReport_FailedReport(t *testing.T) {
	clairReport := &clairclient.IndexReport{
		ManifestHash: "sha256:failed",
		State:        "IndexError",
		Success:      false,
		Err:          "failed to index layer: connection timeout",
	}

	protoReport, err := ToProtoIndexReport(clairReport)
	require.NoError(t, err)
	require.NotNil(t, protoReport)

	assert.Equal(t, "sha256:failed", protoReport.HashId)
	assert.Equal(t, "IndexError", protoReport.State)
	assert.False(t, protoReport.Success)
	assert.Equal(t, "failed to index layer: connection timeout", protoReport.Err)

	// Contents should still be created even for failed reports
	require.NotNil(t, protoReport.Contents)
}
