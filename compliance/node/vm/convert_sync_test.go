package vm

import (
	"testing"

	"github.com/quay/claircore"
	"github.com/quay/claircore/toolkit/types/cpe"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/scannerv4/mappers"
	"github.com/stretchr/testify/require"
)

// TestConversionsSyncWithMappers ensures VM conversion functions produce
// identical output to pkg/scannerv4/mappers to prevent drift.
// This test guards against accidental divergence between the duplicated implementations.
//
// Note: This tests the public API (Contents conversion) since the internal conversion
// functions are private in both packages.
func TestConversionsSyncWithMappers(t *testing.T) {
	digest := claircore.MustParseDigest("sha256:0000000000000000000000000000000000000000000000000000000000000000")

	testIndexReport := &claircore.IndexReport{
		State:   "IndexFinished",
		Success: true,
		Packages: map[string]*claircore.Package{
			"pkg-1": {
				ID:             "pkg-1",
				Name:           "test-pkg",
				Version:        "1.0.0",
				Kind:           "binary",
				PackageDB:      "dpkg",
				Arch:           "amd64",
				Module:         "",
				RepositoryHint: "ubuntu-main",
			},
			"pkg-2": {
				ID:      "pkg-2",
				Name:    "src-pkg",
				Version: "2.0.0",
				Kind:    "source",
				Source: &claircore.Package{
					ID:      "pkg-2-src",
					Name:    "src-pkg",
					Version: "2.0.0",
				},
			},
		},
		Distributions: map[string]*claircore.Distribution{
			"dist-1": {
				ID:              "dist-1",
				DID:             "ubuntu",
				Name:            "Ubuntu",
				Version:         "20.04",
				VersionID:       "20.04",
				VersionCodeName: "focal",
				Arch:            "amd64",
				PrettyName:      "Ubuntu 20.04 LTS",
			},
			"dist-2": {
				ID:         "alpine-1",
				DID:        "alpine",
				Name:       "Alpine Linux",
				Version:    "3.15",
				VersionID:  "", // Alpine doesn't populate VersionID - tests fallback logic
				Arch:       "x86_64",
				PrettyName: "Alpine Linux v3.15",
			},
		},
		Repositories: map[string]*claircore.Repository{
			"repo-1": {
				ID:   "repo-1",
				Name: "ubuntu-main",
				Key:  "Ubuntu",
				URI:  "http://archive.ubuntu.com/ubuntu",
			},
			"repo-2": {
				ID:   "repo-2",
				Name: "rhel-baseos",
				Key:  "rhel-cpe-repository",
				URI:  "https://cdn.redhat.com/content/dist/rhel8/8/x86_64/baseos/os",
				CPE:  cpe.MustUnbind("cpe:2.3:a:redhat:enterprise_linux:8:*:*:*:*:*:*:*"),
			},
		},
		Environments: map[string][]*claircore.Environment{
			"pkg-1": {
				{
					PackageDB:      "dpkg",
					IntroducedIn:   digest,
					DistributionID: "dist-1",
					RepositoryIDs:  []string{"repo-1"},
				},
			},
			"pkg-2": {
				{
					PackageDB:      "rpm",
					IntroducedIn:   digest,
					DistributionID: "dist-2",
					RepositoryIDs:  []string{"repo-2"},
				},
			},
		},
	}

	t.Run("full IndexReport conversion matches", func(t *testing.T) {
		// Convert using VM converter
		vmResult := toProtoV4IndexReport(testIndexReport)
		require.NotNil(t, vmResult)

		// Convert using mappers converter
		mappersResult, err := mappers.ToProtoV4IndexReport(testIndexReport)
		require.NoError(t, err)
		require.NotNil(t, mappersResult)

		// Compare top-level fields
		require.Equal(t, mappersResult.GetState(), vmResult.GetState())
		require.Equal(t, mappersResult.GetSuccess(), vmResult.GetSuccess())
		require.Equal(t, mappersResult.GetErr(), vmResult.GetErr())

		// Compare Contents in detail
		compareContents(t, mappersResult.GetContents(), vmResult.GetContents())
	})

	t.Run("Alpine distribution VersionID fallback", func(t *testing.T) {
		// Specifically test Alpine's missing VersionID fallback
		alpineReport := &claircore.IndexReport{
			Distributions: map[string]*claircore.Distribution{
				"alpine": {
					ID:        "alpine-1",
					DID:       "alpine",
					Version:   "3.15.0",
					VersionID: "", // Alpine doesn't populate this
				},
			},
		}

		vmResult := toProtoV4IndexReport(alpineReport)
		mappersResult, err := mappers.ToProtoV4IndexReport(alpineReport)
		require.NoError(t, err)

		// Both should fall back to Version for Alpine
		vmDist := vmResult.GetContents().GetDistributions()["alpine"]
		mappersDist := mappersResult.GetContents().GetDistributions()["alpine"]

		require.Equal(t, "3.15.0", vmDist.GetVersionId(), "VM converter should use Version for Alpine")
		require.Equal(t, "3.15.0", mappersDist.GetVersionId(), "Mappers converter should use Version for Alpine")
		protoassert.Equal(t, mappersDist, vmDist)
	})
}

func compareContents(t *testing.T, expected, actual *v4.Contents) {
	require.NotNil(t, expected)
	require.NotNil(t, actual)

	// Compare packages
	require.Len(t, actual.GetPackages(), len(expected.GetPackages()))
	for id, expectedPkg := range expected.GetPackages() {
		actualPkg := actual.GetPackages()[id]
		require.NotNil(t, actualPkg, "missing package: %s", id)
		protoassert.Equal(t, expectedPkg, actualPkg, "package %s differs", id)
	}

	// Compare distributions
	require.Len(t, actual.GetDistributions(), len(expected.GetDistributions()))
	for id, expectedDist := range expected.GetDistributions() {
		actualDist := actual.GetDistributions()[id]
		require.NotNil(t, actualDist, "missing distribution: %s", id)
		protoassert.Equal(t, expectedDist, actualDist, "distribution %s differs", id)
	}

	// Compare repositories
	require.Len(t, actual.GetRepositories(), len(expected.GetRepositories()))
	for id, expectedRepo := range expected.GetRepositories() {
		actualRepo := actual.GetRepositories()[id]
		require.NotNil(t, actualRepo, "missing repository: %s", id)
		protoassert.Equal(t, expectedRepo, actualRepo, "repository %s differs", id)
	}

	// Compare environments
	require.Len(t, actual.GetEnvironments(), len(expected.GetEnvironments()))
	for id, expectedEnvList := range expected.GetEnvironments() {
		actualEnvList := actual.GetEnvironments()[id]
		require.NotNil(t, actualEnvList, "missing environment list: %s", id)
		protoassert.Equal(t, expectedEnvList, actualEnvList, "environment %s differs", id)
	}
}
