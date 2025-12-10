package fake

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
	// vmMockDigest may be kept in sync with its copy in pkg/virtualmachine/enricher/enricher_impl.go
	vmMockDigest = "sha256:900dc0ffee900dc0ffee900dc0ffee900dc0ffee900dc0ffee900dc0ffee900d"
)

// numericRegex matches sequences of digits in a version string.
var numericRegex = regexp.MustCompile(`\d+`)

// normalizeRPMVersion parses an RPM version string and returns a 10-element int32 slice.
// Only the first 3 numeric components (major, minor, patch) are extracted since those
// are what matter for vulnerability matching. The rest are zeros.
// The epoch prefix (e.g., "1:") is stripped before parsing.
// Example: "1:1.54.0-3.el9_7" → [1, 54, 0, 0, 0, 0, 0, 0, 0, 0]
func normalizeRPMVersion(version string) []int32 {
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

// reportGenerator generates fake VM index reports using a constant pre-built template
type reportGenerator struct {
	packages     map[string]*v4.Package
	repositories map[string]*v4.Repository
	environments map[string]*v4.Environment_List
}

// selectPackageIndices returns a slice of indices into rhel9Packages based on numRequested.
// - numRequested <= 0: returns all indices sequentially [0, 1, ..., totalAvailable-1]
// - numRequested < totalAvailable: randomly samples numRequested indices
// - numRequested >= totalAvailable: uses all indices, then duplicates randomly to fill
func selectPackageIndices(rng *rand.Rand, numRequested, totalAvailable int) []int {
	switch {
	case numRequested <= 0:
		// Use all packages sequentially
		indices := make([]int, totalAvailable)
		for i := range indices {
			indices[i] = i
		}
		return indices
	case numRequested < totalAvailable:
		// Randomly sample numRequested from available packages
		return rng.Perm(totalAvailable)[:numRequested]
	default:
		// numRequested >= totalAvailable: use all, then duplicate randomly to fill
		indices := make([]int, numRequested)
		for i := 0; i < totalAvailable; i++ {
			indices[i] = i
		}
		for i := totalAvailable; i < numRequested; i++ {
			indices[i] = rng.Intn(totalAvailable)
		}
		return indices
	}
}

// buildRepositories creates real RHEL 9 repositories and synthetic ones if numRepos exceeds available.
// Returns the repositories map and synthetic repo IDs (for assigning duplicated packages).
// If less than `len(rhel9Repositories)==2` are requested, then it always returns the two real repositories.
func buildRepositories(numRepos int) (map[string]*v4.Repository, []string) {
	totalRealRepos := len(rhel9Repositories)
	repoCount := totalRealRepos
	if numRepos > totalRealRepos {
		repoCount = numRepos
	}

	repositories := make(map[string]*v4.Repository, repoCount)

	// Add real repositories first
	for repoID, cpe := range rhel9Repositories {
		repositories[repoID] = &v4.Repository{
			Id:   repoID,
			Name: repoID,
			Uri:  fmt.Sprintf("https://cdn.redhat.com/content/dist/rhel9/%s", repoID),
			Key:  "rhel-cpe-repository", // Required for ClairCore RHEL matching
			Cpe:  cpe,
		}
	}

	// Add synthetic repositories if requested
	var syntheticRepoIDs []string
	for i := totalRealRepos; i < numRepos; i++ {
		repoID := fmt.Sprintf("synthetic-repo-%d", i)
		repositories[repoID] = &v4.Repository{
			Id:   repoID,
			Name: fmt.Sprintf("Synthetic Repository %d", i),
			Uri:  fmt.Sprintf("https://example.com/repos/synthetic-%d", i),
			Key:  fmt.Sprintf("synthetic-%d", i),
			Cpe:  fmt.Sprintf("cpe:2.3:a:example:synthetic_repo:%d:*:*:*:*:*:*:*", i),
		}
		syntheticRepoIDs = append(syntheticRepoIDs, repoID)
	}

	return repositories, syntheticRepoIDs
}

// newReportGenerator creates a report generator using real RHEL 9 package data.
// The numPackages parameter specifies how many packages to include (0 = all available).
// When numPackages < available, packages are randomly sampled.
// When numPackages > available, packages are duplicated to reach the requested count.
// The numRepos parameter specifies how many repositories to include (0 = real repos only).
// When numRepos > available, synthetic repositories are created to reach the requested count.
// The seed parameter controls random selection for reproducibility.
func newReportGenerator(numPackages, numRepos int, seed int64) *reportGenerator {
	rng := rand.New(rand.NewSource(seed))

	totalPkgs := len(rhel9Packages)
	indices := selectPackageIndices(rng, numPackages, totalPkgs)
	repositories, syntheticRepoIDs := buildRepositories(numRepos)

	// Build packages from fixture data using selected indices
	// - Real packages (first totalPkgs) keep their original repo from fixture
	// - Duplicated packages (beyond totalPkgs) are distributed across synthetic repos
	packages := make(map[string]*v4.Package, len(indices))
	environments := make(map[string]*v4.Environment_List, len(indices))

	for i, idx := range indices {
		pkg := rhel9Packages[idx]
		pkgID := fmt.Sprintf("%s-%d", pkg.Name, i)

		var assignedRepoID string
		if i < totalPkgs {
			// Real package: keep original repo from fixture
			assignedRepoID = pkg.Repo
		} else if len(syntheticRepoIDs) > 0 {
			// Duplicated package: assign to synthetic repos round-robin
			assignedRepoID = syntheticRepoIDs[(i-totalPkgs)%len(syntheticRepoIDs)]
		} else {
			// No synthetic repos available, use original repo
			assignedRepoID = pkg.Repo
		}
		repoCPE := repositories[assignedRepoID].GetCpe()

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
				V:    normalizeRPMVersion(pkg.Version),
			},
		}

		// Environment maps package ID to its repository
		environments[pkgID] = &v4.Environment_List{
			Environments: []*v4.Environment{
				{
					PackageDb:     "sqlite:usr/share/rpm",
					IntroducedIn:  "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					RepositoryIds: []string{assignedRepoID},
				},
			},
		}
	}

	return &reportGenerator{
		repositories: repositories,
		packages:     packages,
		environments: environments,
	}
}

// generateFakeIndexReport creates a fake VM index report using the provided generator.
func generateFakeIndexReport(gen *reportGenerator, vsockCID uint32) *v1.IndexReport {
	return &v1.IndexReport{
		VsockCid: fmt.Sprintf("%d", vsockCID),
		IndexV4: &v4.IndexReport{
			HashId:  vmMockDigest,
			State:   "IndexFinished",
			Success: true,
			Contents: &v4.Contents{
				Packages:     gen.packages,
				Repositories: gen.repositories,
				Environments: gen.environments,
			},
		},
	}
}
