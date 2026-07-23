package cmd

import (
	"errors"
	"io/fs"
	"maps"
	"slices"

	"github.com/stackrox/rox/compliance/virtualmachines/roxagent/internal/hostprobe"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
)

// logFilesystemDiagnostics logs a compact summary of host filesystem facts
// (DNF version, entitlement certificates, repo files) that are useful for
// debugging scan issues, regardless of whether the report ends up empty.
//
// DNF version and entitlement status are read from d, the DiscoveredData
// scanWithDiagnostics already computed for this same hostPath - reprobing
// them here would just repeat, on every scan and rescan, filesystem checks
// discovery.DiscoverVMData already did. Repo-file presence is the one fact
// still probed directly: DiscoveredData only carries a found/not-found
// flag, not the reason (missing dir vs. unreadable vs. empty) that
// logRepoError distinguishes for debugging "0 repositories" issues.
func logFilesystemDiagnostics(hostPath string, d *v1.DiscoveredData) {
	switch {
	case slices.Contains(d.GetDnfStatus(), v1.DnfStatusFlag_DNF_V5_HISTORY_DB_FOUND):
		log.Info("DNF history DB (v5) found")
	case slices.Contains(d.GetDnfStatus(), v1.DnfStatusFlag_DNF_V4_HISTORY_DB_FOUND):
		log.Info("DNF history DB (v4) found")
	default:
		log.Warn("DNF history DB not found!")
	}

	switch d.GetActivationStatus() {
	case v1.ActivationStatus_ACTIVE:
		log.Info("Entitlement certificates found")
	case v1.ActivationStatus_INACTIVE:
		log.Warn("Entitlement certificates not found")
	default:
		log.Warn("Entitlement certificate status could not be determined")
	}

	allReposDirs := append(slices.Clone(hostprobe.DNF4ReposDirs), hostprobe.DNF5ReposDirPath)
	hasRepo, err := hostprobe.HasAnyRepoFileAt(hostPath, allReposDirs)
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
