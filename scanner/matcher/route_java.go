package matcher

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/quay/claircore"
	"github.com/quay/claircore/java"
	rheljava "github.com/quay/claircore/rhel/java"
)

// routeRedHatJavaToOSV makes a matching-only copy of an index report, replacing
// Red Hat product repositories on Java SBOM packages with Maven. The original
// report remains suitable for storage and for returning to Scanner clients.
func routeRedHatJavaToOSV(ir *claircore.IndexReport) *claircore.IndexReport {
	targets := make(map[string][]int)
	for pkgID, pkg := range ir.Packages {
		if pkg == nil || !strings.HasPrefix(pkg.PackageDB, "sbom:") {
			continue
		}
		for i, env := range ir.Environments[pkgID] {
			if env == nil {
				continue
			}
			for _, repoID := range env.RepositoryIDs {
				if repo := ir.Repositories[repoID]; repo != nil && repo.Key == rheljava.RepositoryKey {
					targets[pkgID] = append(targets[pkgID], i)
					break
				}
			}
		}
	}
	if len(targets) == 0 {
		return ir
	}

	copy := *ir
	copy.Repositories = maps.Clone(ir.Repositories)
	copy.Environments = maps.Clone(ir.Environments)
	mavenID := mavenRepositoryID(copy.Repositories)
	for pkgID, indices := range targets {
		envs := slices.Clone(ir.Environments[pkgID])
		for _, i := range indices {
			env := *envs[i]
			env.RepositoryIDs = []string{mavenID}
			envs[i] = &env
		}
		copy.Environments[pkgID] = envs
	}
	return &copy
}

func mavenRepositoryID(repos map[string]*claircore.Repository) string {
	var existingID string
	for id, repo := range repos {
		if repo != nil && repo.Name == java.Repository.Name && repo.Key == "" &&
			(existingID == "" || id < existingID) {
			existingID = id
		}
	}
	if existingID != "" {
		return existingID
	}

	const syntheticID = "stackrox-osv-maven"
	id := syntheticID
	for n := 1; ; n++ {
		if _, exists := repos[id]; !exists {
			break
		}
		id = fmt.Sprintf("%s-%d", syntheticID, n)
	}
	repo := java.Repository
	repo.ID = id
	repos[id] = &repo
	return id
}
