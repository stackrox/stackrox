package operatorbundle

import (
	"sort"
)

// ComputeDiff pairs installed and candidate images by repository name and classifies their
// CVEs. For a repository present in both bundles, each CVE is fixed (installed only), still
// active (both), or new (candidate only). A repository present in only one bundle yields an
// ImageDiff with ImageAdded or ImageRemoved status.
//
// The returned diffs are sorted by repository for deterministic output.
func ComputeDiff(installed, candidate []ImageCVEs) []ImageDiff {
	installedByRepo := indexByRepository(installed)
	candidateByRepo := indexByRepository(candidate)

	repos := unionRepositories(installedByRepo, candidateByRepo)

	diffs := make([]ImageDiff, 0, len(repos))
	for _, repo := range repos {
		inst, hasInst := installedByRepo[repo]
		cand, hasCand := candidateByRepo[repo]

		switch {
		case hasInst && hasCand:
			diffs = append(diffs, pairedDiff(repo, inst, cand))
		case hasCand:
			// Repository only in the candidate bundle: every CVE is newly introduced.
			diffs = append(diffs, ImageDiff{
				Repository:      repo,
				Status:          ImageAdded,
				CandidateDigest: cand.Digest,
				New:             sortedCVEs(cand.CVEs),
			})
		default:
			// Repository only in the installed bundle: every CVE goes away with the upgrade.
			diffs = append(diffs, ImageDiff{
				Repository:      repo,
				Status:          ImageRemoved,
				InstalledDigest: inst.Digest,
				Fixed:           sortedCVEs(inst.CVEs),
			})
		}
	}
	return diffs
}

func pairedDiff(repo string, installed, candidate ImageCVEs) ImageDiff {
	installedCVEs := indexCVEs(installed.CVEs)
	candidateCVEs := indexCVEs(candidate.CVEs)

	diff := ImageDiff{
		Repository:      repo,
		Status:          ImagePaired,
		InstalledDigest: installed.Digest,
		CandidateDigest: candidate.Digest,
	}
	for id, cve := range installedCVEs {
		if _, stillPresent := candidateCVEs[id]; stillPresent {
			diff.StillActive = append(diff.StillActive, cve)
		} else {
			diff.Fixed = append(diff.Fixed, cve)
		}
	}
	for id, cve := range candidateCVEs {
		if _, existed := installedCVEs[id]; !existed {
			diff.New = append(diff.New, cve)
		}
	}
	sortCVEs(diff.Fixed)
	sortCVEs(diff.StillActive)
	sortCVEs(diff.New)
	return diff
}

// indexByRepository indexes images by repository. If multiple images share a repository the
// last one wins; bundles are not expected to contain duplicate repositories.
func indexByRepository(images []ImageCVEs) map[string]ImageCVEs {
	byRepo := make(map[string]ImageCVEs, len(images))
	for _, img := range images {
		byRepo[img.Repository] = img
	}
	return byRepo
}

// indexCVEs indexes CVEs by ID. If a CVE ID appears more than once the last one wins.
func indexCVEs(cves []CVE) map[string]CVE {
	byID := make(map[string]CVE, len(cves))
	for _, c := range cves {
		byID[c.ID] = c
	}
	return byID
}

func unionRepositories(a, b map[string]ImageCVEs) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	for repo := range a {
		seen[repo] = struct{}{}
	}
	for repo := range b {
		seen[repo] = struct{}{}
	}
	repos := make([]string, 0, len(seen))
	for repo := range seen {
		repos = append(repos, repo)
	}
	sort.Strings(repos)
	return repos
}

func sortedCVEs(cves []CVE) []CVE {
	out := make([]CVE, len(cves))
	copy(out, cves)
	sortCVEs(out)
	return out
}

// sortCVEs orders CVEs by descending severity then ascending ID for stable, useful output.
func sortCVEs(cves []CVE) {
	sort.Slice(cves, func(i, j int) bool {
		if cves[i].Severity != cves[j].Severity {
			return cves[i].Severity > cves[j].Severity
		}
		return cves[i].ID < cves[j].ID
	})
}
