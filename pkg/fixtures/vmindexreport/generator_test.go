package vmindexreport

import (
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
		"should duplicate packages when numPackages > available": {
			numPackages:      totalAvailable + 100,
			expectedPkgCount: totalAvailable + 100,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			gen := NewGeneratorWithSeed(tt.numPackages, 42)

			assert.Equal(t, tt.expectedPkgCount, gen.NumPackages(), "package count mismatch")
			assert.Equal(t, 3, gen.NumRepositories(), "should always have 3 real repositories")
		})
	}
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

func TestNewGeneratorWithSeed_DifferentSeeds(t *testing.T) {
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

	// Different seeds should produce different package selection (with high probability)
	// The symmetric difference should be non-empty
	diff := names1.Difference(names2).Union(names2.Difference(names1))
	assert.NotEmpty(t, diff, "different seeds should produce different package selections")
}

func TestGenerateV4IndexReport(t *testing.T) {
	gen := NewGeneratorWithSeed(10, 42)
	report := gen.GenerateV4IndexReport()

	assert.Equal(t, MockDigest, report.GetHashId(), "HashId should match MockDigest")
	assert.Equal(t, "IndexFinished", report.GetState(), "State should be IndexFinished")
	assert.True(t, report.GetSuccess(), "Success should be true")

	require.NotNil(t, report.GetContents(), "Contents should not be nil")
	assert.Len(t, report.GetContents().GetPackages(), 10, "should have 10 packages")
	assert.Len(t, report.GetContents().GetRepositories(), 3, "should have 3 repositories")
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
	assert.Len(t, report.GetContents().GetRepositories(), 3, "should still have 3 repositories even with 0 packages")
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
