package vmindexreport

import (
	"math/rand"
	"testing"

	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGeneratorWithSeed(t *testing.T) {
	totalAvailable := len(packagesFixture)

	tests := map[string]struct {
		numPackages      int
		expectedPkgCount int
	}{
		"should return empty packages when numPackages is 0": {
			numPackages:      0,
			expectedPkgCount: 0,
		},
		"should sample packages when numPackages < available": {
			numPackages:      10,
			expectedPkgCount: 10,
		},
		"should return all packages when numPackages equals available": {
			numPackages:      totalAvailable,
			expectedPkgCount: totalAvailable,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			gen := NewGeneratorWithSeed(tt.numPackages, 42)

			assert.Equal(t, tt.expectedPkgCount, gen.NumPackages(), "package count mismatch")
			assert.Equal(t, len(repoToCPEMapping), gen.NumRepositories(), "repository count mismatch")
		})
	}
}

func TestSelectPackages_BaseAndExtras(t *testing.T) {
	numRequested := baseRHELPackageCount + 5
	selected := selectPackages(rand.New(rand.NewSource(99)), numRequested)

	require.Len(t, selected, numRequested)

	// First baseRHELPackageCount should match the base fixture exactly.
	for i := 0; i < baseRHELPackageCount; i++ {
		assert.Equal(t, basePackagesFixture[i], selected[i], "base package mismatch at %d", i)
	}

	// Extras should not be part of the base set.
	exclude := make(map[string]struct{}, len(basePackagesFixture))
	for _, p := range basePackagesFixture {
		exclude[p.Name+p.Version+p.Repo] = struct{}{}
	}
	seen := make(map[string]struct{})
	for _, p := range selected[baseRHELPackageCount:] {
		key := p.Name + p.Version + p.Repo
		if _, ok := exclude[key]; ok {
			t.Fatalf("extra package %s is part of base set", key)
		}
		if _, ok := seen[key]; ok {
			t.Fatalf("duplicate extra package %s", key)
		}
		seen[key] = struct{}{}
	}
}

func TestNewGeneratorWithSeed_PanicsWhenRequestingTooManyPackages(t *testing.T) {
	exclude := make(map[string]struct{}, len(basePackagesFixture))
	for _, p := range basePackagesFixture {
		exclude[p.Name+p.Version+p.Repo] = struct{}{}
	}
	extras := 0
	for _, p := range packagesFixture {
		if _, ok := exclude[p.Name+p.Version+p.Repo]; ok {
			continue
		}
		extras++
	}
	totalAvailable := baseRHELPackageCount + extras

	assert.Panics(t, func() {
		NewGeneratorWithSeed(totalAvailable+1, 42)
	}, "should panic when numPackages exceeds available packages")
}

func TestNewGeneratorWithSeed_PanicsOnNegativePackages(t *testing.T) {
	assert.Panics(t, func() {
		NewGeneratorWithSeed(-1, 42)
	}, "should panic when numPackages is negative")
}

func TestNewGeneratorWithSeed_Reproducibility(t *testing.T) {
	gen1 := NewGeneratorWithSeed(50, 123)
	gen2 := NewGeneratorWithSeed(50, 123)

	report1 := gen1.GenerateV4IndexReport()
	report2 := gen2.GenerateV4IndexReport()

	// Same seed should produce identical package selection
	assert.Equal(t, len(report1.GetContents().GetPackages()), len(report2.GetContents().GetPackages()))

	for pkgID, pkg1 := range report1.GetContents().GetPackages() {
		pkg2, exists := report2.GetContents().GetPackages()[pkgID]
		require.True(t, exists, "package %s should exist in both reports", pkgID)
		assert.Equal(t, pkg1.GetName(), pkg2.GetName(), "package names should match")
		assert.Equal(t, pkg1.GetVersion(), pkg2.GetVersion(), "package versions should match")
	}
}

func TestNewGeneratorWithSeed_DeterministicPackageSelection(t *testing.T) {
	// Test that we always get the same first N packages regardless of seed
	gen1 := NewGeneratorWithSeed(10, 111)
	gen2 := NewGeneratorWithSeed(10, 222)

	report1 := gen1.GenerateV4IndexReport()
	report2 := gen2.GenerateV4IndexReport()

	// Collect package names into sets
	names1 := set.NewStringSet()
	names2 := set.NewStringSet()
	for _, pkg := range report1.GetContents().GetPackages() {
		names1.Add(pkg.GetName())
	}
	for _, pkg := range report2.GetContents().GetPackages() {
		names2.Add(pkg.GetName())
	}

	// Different seeds should now produce SAME package selection (deterministic)
	assert.Equal(t, names1, names2, "different seeds should produce identical package selections")

	// Verify we get the first N packages from the fixture
	expectedNames := set.NewStringSet()
	for i := 0; i < 10; i++ {
		expectedNames.Add(basePackagesFixture[i].Name)
	}
	assert.Equal(t, expectedNames, names1, "should contain first 10 packages from fixture")
}

func TestNewGeneratorWithSeed_AlwaysProducesSamePackageSet(t *testing.T) {
	// Test the behavior described: requesting 5 packages always gives first 5,
	// requesting 7 packages always gives first 7, etc.
	tests := []struct {
		name             string
		numPackages      int
		expectedPackages []string
	}{
		{
			name:        "5 packages should always be first 5 from fixture",
			numPackages: 5,
			expectedPackages: []string{
				"NetworkManager",
				"NetworkManager-libnm",
				"NetworkManager-team",
				"NetworkManager-tui",
				"PackageKit",
			},
		},
		{
			name:        "7 packages should always be first 7 from fixture",
			numPackages: 7,
			expectedPackages: []string{
				"NetworkManager",
				"NetworkManager-libnm",
				"NetworkManager-team",
				"NetworkManager-tui",
				"PackageKit",
				"PackageKit-glib",
				"abattis-cantarell-fonts",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := NewGeneratorWithSeed(tt.numPackages, 42)
			report := gen.GenerateV4IndexReport()

			actualNames := set.NewStringSet()
			for _, pkg := range report.GetContents().GetPackages() {
				actualNames.Add(pkg.GetName())
			}

			expectedNames := set.NewStringSet()
			expectedNames.AddAll(tt.expectedPackages...)

			assert.Equal(t, expectedNames, actualNames, "package set mismatch")
			assert.Equal(t, tt.numPackages, actualNames.Cardinality(), "package count mismatch")
		})
	}
}

func TestGenerateV4IndexReport(t *testing.T) {
	gen := NewGeneratorWithSeed(10, 42)
	report := gen.GenerateV4IndexReport()

	assert.Equal(t, MockDigest, report.GetHashId(), "HashId should match MockDigest")
	assert.Equal(t, "IndexFinished", report.GetState(), "State should be IndexFinished")
	assert.True(t, report.GetSuccess(), "Success should be true")

	require.NotNil(t, report.GetContents(), "Contents should not be nil")
	assert.Len(t, report.GetContents().GetPackages(), 10, "should have 10 packages")
	assert.Len(t, report.GetContents().GetRepositories(), len(repoToCPEMapping), "repository count mismatch")
	assert.Len(t, report.GetContents().GetEnvironments(), 10, "should have 10 environments")
}

func TestGenerateV4IndexReport_ZeroPackages(t *testing.T) {
	gen := NewGeneratorWithSeed(0, 42)
	report := gen.GenerateV4IndexReport()

	// Report metadata should be unchanged
	assert.Equal(t, MockDigest, report.GetHashId(), "HashId should match MockDigest")
	assert.Equal(t, "IndexFinished", report.GetState(), "State should be IndexFinished")
	assert.True(t, report.GetSuccess(), "Success should be true")

	require.NotNil(t, report.GetContents(), "Contents should not be nil")
	// Packages and Environments should be empty when numPackages is 0
	assert.Empty(t, report.GetContents().GetPackages(), "Packages should be empty when numPackages is 0")
	assert.Empty(t, report.GetContents().GetEnvironments(), "Environments should be empty when numPackages is 0")
	// Repositories should still have the three real repos
	assert.Len(t, report.GetContents().GetRepositories(), len(repoToCPEMapping), "repository count mismatch even with 0 packages")
}

func TestGenerateV4IndexReport_PackagesHaveValidCPEs(t *testing.T) {
	gen := NewGeneratorWithSeed(20, 42)
	report := gen.GenerateV4IndexReport()

	for pkgID, pkg := range report.GetContents().GetPackages() {
		assert.NotEmpty(t, pkg.GetCpe(), "package %s should have a CPE", pkgID)
		assert.Contains(t, pkg.GetCpe(), "cpe:2.3:", "package %s CPE should be in 2.3 format", pkgID)

		if pkg.GetSource() != nil {
			assert.NotEmpty(t, pkg.GetSource().GetCpe(), "source package %s should have a CPE", pkgID)
		}
	}
}

func TestNormalizeRPMVersion(t *testing.T) {
	tests := map[string]struct {
		version  string
		expected []int32
	}{
		"should parse simple version": {
			version:  "1.2.3",
			expected: []int32{1, 2, 3, 0, 0, 0, 0, 0, 0, 0},
		},
		"should strip epoch prefix": {
			version:  "1:1.54.0-3.el9_7",
			expected: []int32{1, 54, 0, 0, 0, 0, 0, 0, 0, 0},
		},
		"should handle version with release suffix": {
			version:  "2.35.2-67.el9",
			expected: []int32{2, 35, 2, 0, 0, 0, 0, 0, 0, 0},
		},
		"should handle single component version": {
			version:  "11",
			expected: []int32{11, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		},
		"should handle two component version": {
			version:  "3.6",
			expected: []int32{3, 6, 0, 0, 0, 0, 0, 0, 0, 0},
		},
		"should handle empty version": {
			version:  "",
			expected: []int32{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		},
		"should handle version with no numbers": {
			version:  "alpha",
			expected: []int32{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := NormalizeRPMVersion(tt.version)
			assert.Equal(t, tt.expected, result)
		})
	}
}
