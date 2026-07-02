package cmd

import (
	"errors"
	"io/fs"
	"maps"
	"os"
	"slices"

	"github.com/stackrox/rox/compliance/virtualmachines/roxagent/internal/hostprobe"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
)

// logFilesystemDiagnostics logs a compact summary of host filesystem facts
// (DNF version, entitlement certificates, repo files) that are useful for
// debugging scan issues, regardless of whether the report ends up empty.
func logFilesystemDiagnostics(hostPath string) {
	switch hostprobe.DetectDNFVersion(hostPath) {
	case hostprobe.DNFVersion5:
		log.Info("DNF history DB (v5) found")
	case hostprobe.DNFVersion4:
		log.Info("DNF history DB (v4) found")
	default:
		log.Warn("DNF history DB not found!")
	}

	hasEntitlement, err := hostprobe.HasEntitlementCertKeyPair(hostPath)
	if err != nil {
		log.Warnf("Entitlement certificates not found: %v", err)
	} else if !hasEntitlement {
		log.Warn("Entitlement certificates not found")
	} else {
		log.Info("Entitlement certificates found")
	}

	allReposDirs := append(slices.Clone(hostprobe.DNF4ReposDirs), hostprobe.DNF5ReposDirPath)
	hasRepo, err := hostprobe.HasAnyRepoFile(os.DirFS(hostPath), allReposDirs)
	if err != nil {
		logRepoError(err)
		return
	}
	if !hasRepo {
		log.Info("Repo dirs are present but contain 0 .repo files")
		return
	}
	log.Info("Repo dirs contain .repo files")
}

func logRepoError(err error) {
	switch {
	case errors.Is(err, fs.ErrNotExist):
		log.Info("No repo directories found")
	case errors.Is(err, fs.ErrPermission):
		log.Warnf("Repo directories are not readable: %v", err)
	default:
		log.Infof("Repo directories are unavailable: %v", err)
	}
}

// logIndexReportDiagnostics logs a summary of the freshly generated index
// report so that "0 packages scanned" issues can be diagnosed from agent
// logs alone. Repository/distribution listings are truncated to the first
// 10 entries to keep logs bounded on hosts with many repos.
func logIndexReportDiagnostics(report *v4.IndexReport) {
	const maxListedEntries = 10

	contents := report.GetContents()

	numPkgs := len(contents.GetPackages())
	numRepos := len(contents.GetRepositories())
	numDists := len(contents.GetDistributions())
	numEnvs := len(contents.GetEnvironments())

	log.Infof("Index report summary: packages=%d, repositories=%d, distributions=%d, environments=%d",
		numPkgs, numRepos, numDists, numEnvs)

	repos := contents.GetRepositories()
	for repoIdx, id := range slices.Sorted(maps.Keys(repos)) {
		repo := repos[id]
		log.Infof("Repository (%d of %d) id=%q name=%q key=%q cpe=%q",
			repoIdx+1, numRepos, id, repo.GetName(), repo.GetKey(), repo.GetCpe())
		if repoIdx+1 >= maxListedEntries && numRepos > maxListedEntries {
			log.Infof("  (%d more truncated for brevity)", numRepos-maxListedEntries)
			break
		}
	}
	dists := contents.GetDistributions()
	for distIdx, id := range slices.Sorted(maps.Keys(dists)) {
		dist := dists[id]
		log.Infof("Distribution (%d of %d) id=%s name=%q version=%q cpe=%q did=%q",
			distIdx+1, numDists, id, dist.GetName(), dist.GetVersion(), dist.GetCpe(), dist.GetDid())
		if distIdx+1 >= maxListedEntries && numDists > maxListedEntries {
			log.Infof("  (%d more truncated for brevity)", numDists-maxListedEntries)
			break
		}
	}

	if numRepos == 0 {
		log.Warn("Index report contains 0 repositories. Packages will be marked as UNSCANNED.")
	}
}
