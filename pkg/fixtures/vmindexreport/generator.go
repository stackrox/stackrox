// Package vmindexreport provides utilities for generating fake VM index reports
// for testing and load testing purposes.
package vmindexreport

import (
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
)

const (
	// MockDigest is kept in sync with pkg/virtualmachine/enricher/enricher_impl.go
	MockDigest = "sha256:900dc0ffee900dc0ffee900dc0ffee900dc0ffee900dc0ffee900dc0ffee900d"
)

// numericRegex matches sequences of digits in a version string.
var numericRegex = regexp.MustCompile(`\d+`)

// NormalizeRPMVersion parses an RPM version string and returns a 10-element int32 slice.
// Only the first 3 numeric components (major, minor, patch) are extracted since those
// are what matter for vulnerability matching. The rest are zeros.
// The epoch prefix (e.g., "1:") is stripped before parsing.
// Example: "1:1.54.0-3.el9_7" → [1, 54, 0, 0, 0, 0, 0, 0, 0, 0]
func NormalizeRPMVersion(version string) []int32 {
	result := make([]int32, 10)

	// Strip epoch prefix if present (e.g., "1:" in "1:1.54.0-3.el9_7")
	if idx := strings.Index(version, ":"); idx != -1 {
		version = version[idx+1:]
	}

	// Extract first 3 numeric sequences from the version string
	matches := numericRegex.FindAllString(version, 3)

	for i, match := range matches {
		if val, err := strconv.ParseInt(match, 10, 32); err == nil {
			result[i] = int32(val)
		}
	}

	return result
}

// Generator generates fake VM index reports using a constant pre-built template.
type Generator struct {
	packages     map[string]*v4.Package
	repositories map[string]*v4.Repository
	environments map[string]*v4.Environment_List
}

// selectPackageIndices returns a slice of indices into packagesFixture based on numRequested.
// - numRequested <= 0: returns empty slice (no packages)
// - numRequested < totalAvailable: randomly samples numRequested indices
// - numRequested >= totalAvailable: uses all indices, then duplicates randomly to fill
func selectPackageIndices(rng *rand.Rand, numRequested, totalAvailable int) []int {
	switch {
	case numRequested <= 0:
		// Return empty slice (no packages)
		return []int{}
	case numRequested < totalAvailable:
		// Randomly sample numRequested from available packages
		return rng.Perm(totalAvailable)[:numRequested]
	default:
		// numRequested >= totalAvailable: use all, then duplicate randomly to fill
		indices := make([]int, numRequested)
		for i := range totalAvailable {
			indices[i] = i
		}
		for i := totalAvailable; i < numRequested; i++ {
			indices[i] = rng.Intn(totalAvailable)
		}
		return indices
	}
}

// buildRepositories creates the two real RHEL repositories from the fixture.
func buildRepositories() map[string]*v4.Repository {
	repositories := make(map[string]*v4.Repository, len(repoToCPEMapping))

	for repoID, cpe := range repoToCPEMapping {
		repositories[repoID] = &v4.Repository{
			Id:   repoID,
			Name: repoID,
			Uri:  fmt.Sprintf("https://cdn.redhat.com/content/dist/rhel9/%s", repoID),
			Key:  "rhel-cpe-repository", // Required for ClairCore RHEL matching
			Cpe:  cpe,
		}
	}

	return repositories
}

// NewGeneratorWithSeed creates a new Generator with a specific random seed.
// The numPackages parameter specifies how many packages to include.
// When numPackages == 0, no packages are included (empty report).
// When numPackages < available, packages are randomly sampled.
// When numPackages > available, packages are duplicated to reach the requested count.
// All packages use the two real RHEL repositories from the fixture.
// The seed parameter controls random selection for reproducibility.
func NewGeneratorWithSeed(numPackages int, seed int64) *Generator {
	if numPackages < 0 {
		panic(fmt.Sprintf("numPackages must be non-negative, got %d", numPackages))
	}
	rng := rand.New(rand.NewSource(seed))

	totalPkgs := len(packagesFixture)
	indices := selectPackageIndices(rng, numPackages, totalPkgs)
	repositories := buildRepositories()

	// Build packages from fixture data using selected indices
	// All packages use their original repo from the fixture
	packages := make(map[string]*v4.Package, len(indices))
	environments := make(map[string]*v4.Environment_List, len(indices))

	for i, idx := range indices {
		pkg := packagesFixture[idx]
		pkgID := fmt.Sprintf("%s-%d", pkg.Name, i)

		// All packages use their original repo from the fixture
		repoCPE := repositories[pkg.Repo].GetCpe()

		packages[pkgID] = &v4.Package{
			Id:             pkgID,
			Name:           pkg.Name,
			Version:        pkg.Version,
			Kind:           "binary",
			Arch:           "x86_64",
			RepositoryHint: "hash:sha256:f52ca767328e6919ec11a1da654e92743587bd3c008f0731f8c4de3af19c1830|key:199e2f91fd431d51",
			Cpe:            repoCPE,
			PackageDb:      "sqlite:usr/share/rpm",
			Source: &v4.Package{
				Id:      pkgID + "-src",
				Name:    pkg.Name,
				Version: pkg.Version,
				Kind:    "source",
				Cpe:     repoCPE,
			},
			NormalizedVersion: &v4.NormalizedVersion{
				Kind: "rpm",
				V:    NormalizeRPMVersion(pkg.Version),
			},
		}

		// Environment maps package ID to its repository
		environments[pkgID] = &v4.Environment_List{
			Environments: []*v4.Environment{
				{
					PackageDb:     "sqlite:usr/share/rpm",
					IntroducedIn:  "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					RepositoryIds: []string{pkg.Repo},
				},
			},
		}
	}

	return &Generator{
		repositories: repositories,
		packages:     packages,
		environments: environments,
	}
}

// GenerateV1IndexReport creates a fake v1.IndexReport (used by Sensor for sending to Central).
func (g *Generator) GenerateV1IndexReport(vsockCID uint32) *v1.IndexReport {
	return &v1.IndexReport{
		VsockCid: fmt.Sprintf("%d", vsockCID),
		IndexV4:  g.GenerateV4IndexReport(),
	}
}

// GenerateV4IndexReport creates a fake v4.IndexReport (used by Scanner V4).
func (g *Generator) GenerateV4IndexReport() *v4.IndexReport {
	return &v4.IndexReport{
		HashId:  MockDigest,
		State:   "IndexFinished",
		Success: true,
		Contents: &v4.Contents{
			Packages:     g.packages,
			Repositories: g.repositories,
			Environments: g.environments,
		},
	}
}

// NumPackages returns the number of packages in the generator.
func (g *Generator) NumPackages() int {
	return len(g.packages)
}

// NumRepositories returns the number of repositories in the generator.
func (g *Generator) NumRepositories() int {
	return len(g.repositories)
}
