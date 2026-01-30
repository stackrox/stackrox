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

const baseRHELPackageCount = 508

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

func packageKey(p PackageFixture) string {
	return p.Name + "|" + p.Version + "|" + p.Repo
}

// selectPackages returns concrete fixtures according to the rules:
//   - numRequested <= 0: empty selection.
//   - numRequested <= baseRHELPackageCount: take the first N from the original RHEL9 base fixture.
//   - numRequested > baseRHELPackageCount: take all base RHEL9 packages, then sample
//     (deterministically via rng) from the remainder of the full fixture, excluding base entries.
//   - Panics if the request exceeds available packages.
func selectPackages(rng *rand.Rand, numRequested int) []PackageFixture {
	if numRequested <= 0 {
		return nil
	}

	if numRequested <= baseRHELPackageCount {
		return basePackagesFixture[:numRequested]
	}

	extrasNeeded := numRequested - baseRHELPackageCount

	// Build exclusion set for fast lookup.
	exclude := make(map[string]struct{}, len(basePackagesFixture))
	for _, p := range basePackagesFixture {
		exclude[packageKey(p)] = struct{}{}
	}

	extrasPool := make([]PackageFixture, 0, len(packagesFixture))
	for _, p := range packagesFixture {
		if _, ok := exclude[packageKey(p)]; ok {
			continue
		}
		extrasPool = append(extrasPool, p)
	}

	if extrasNeeded > len(extrasPool) {
		panic(fmt.Sprintf("numPackages must be <= %d, got %d", baseRHELPackageCount+len(extrasPool), numRequested))
	}

	perm := rng.Perm(len(extrasPool))

	selected := make([]PackageFixture, 0, numRequested)
	selected = append(selected, basePackagesFixture...)
	for i := 0; i < extrasNeeded; i++ {
		selected = append(selected, extrasPool[perm[i]])
	}

	return selected
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

// NewGeneratorWithSeed creates a new Generator with deterministic package selection.
// The numPackages parameter specifies how many packages to include.
//   - numPackages == 0: no packages are included (empty report).
//   - numPackages <= baseRHELPackageCount: use the first numPackages from the original RHEL9 fixture.
//   - numPackages > baseRHELPackageCount: include the first baseRHELPackageCount packages,
//     then sample the remaining from the rest of the fixture using the provided seed.
//   - numPackages > available: panic.
//
// All packages use the two real RHEL repositories from the fixture.
func NewGeneratorWithSeed(numPackages int, seed int64) *Generator {
	if numPackages < 0 {
		panic(fmt.Sprintf("numPackages must be non-negative, got %d", numPackages))
	}
	rng := rand.New(rand.NewSource(seed))

	selected := selectPackages(rng, numPackages)
	repositories := buildRepositories()

	packages := make(map[string]*v4.Package, len(selected))
	environments := make(map[string]*v4.Environment_List, len(selected))

	for i, pkg := range selected {
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

// NewGeneratorWithSpecificPackage creates a new Generator where all packages use a specific package name.
// This is useful for load testing with packages that have specific vulnerability characteristics.
// packageName must exist in packagesFixture. Common test packages:
//   - "vim-minimal": High vulnerability count (for stress testing)
//   - "basesystem": Zero vulnerabilities (metapackage, no code)
//   - "filesystem": Zero vulnerabilities (directory structure only)
func NewGeneratorWithSpecificPackage(packageName string, numPackages int) *Generator {
	if numPackages < 0 {
		panic(fmt.Sprintf("numPackages must be non-negative, got %d", numPackages))
	}
	if numPackages == 0 {
		return &Generator{
			repositories: buildRepositories(),
			packages:     make(map[string]*v4.Package),
			environments: make(map[string]*v4.Environment_List),
		}
	}

	// Find package in the fixture
	var targetPkg *PackageFixture
	for i := range packagesFixture {
		if packagesFixture[i].Name == packageName {
			targetPkg = &packagesFixture[i]
			break
		}
	}
	if targetPkg == nil {
		panic(fmt.Sprintf("package %q not found in packagesFixture", packageName))
	}

	repositories := buildRepositories()
	packages := make(map[string]*v4.Package, numPackages)
	environments := make(map[string]*v4.Environment_List, numPackages)

	// Create numPackages instances of the specified package
	repoCPE := repositories[targetPkg.Repo].GetCpe()
	for i := 0; i < numPackages; i++ {
		pkgID := fmt.Sprintf("%s-%d", targetPkg.Name, i)

		packages[pkgID] = &v4.Package{
			Id:             pkgID,
			Name:           targetPkg.Name,
			Version:        targetPkg.Version,
			Kind:           "binary",
			Arch:           "x86_64",
			RepositoryHint: "hash:sha256:f52ca767328e6919ec11a1da654e92743587bd3c008f0731f8c4de3af19c1830|key:199e2f91fd431d51",
			Cpe:            repoCPE,
			PackageDb:      "sqlite:usr/share/rpm",
			Source: &v4.Package{
				Id:      pkgID + "-src",
				Name:    targetPkg.Name,
				Version: targetPkg.Version,
				Kind:    "source",
				Cpe:     repoCPE,
			},
			NormalizedVersion: &v4.NormalizedVersion{
				Kind: "rpm",
				V:    NormalizeRPMVersion(targetPkg.Version),
			},
		}

		environments[pkgID] = &v4.Environment_List{
			Environments: []*v4.Environment{
				{
					PackageDb:     "sqlite:usr/share/rpm",
					IntroducedIn:  "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					RepositoryIds: []string{targetPkg.Repo},
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

// NewGeneratorWithVimMinimal creates a new Generator where all packages are vim-minimal.
// This is useful for load testing vulnerability processing since vim-minimal has many CVEs.
// The numPackages parameter specifies how many vim-minimal package instances to include.
func NewGeneratorWithVimMinimal(numPackages int) *Generator {
	return NewGeneratorWithSpecificPackage("vim-minimal", numPackages)
}
