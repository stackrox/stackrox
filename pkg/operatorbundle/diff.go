package operatorbundle

import (
	"sort"
)

// ComputeDiff pairs installed and candidate images by their (repository, name) key and
// classifies their CVEs. For an image present in both bundles, each CVE is fixed (installed
// only), still active (both), or new (candidate only). An image present in only one bundle
// yields an ImageDiff with ImageAdded or ImageRemoved status.
//
// Pairing keys on (repository, name) rather than repository alone because multi-version
// bundles ship many images under one repository (e.g. 12 istio-cni-rhel9 variants), each with
// a distinct related_images name; keying on repository alone would collapse them and diff the
// wrong pair. See mt_nogit_image_remed/phase1_validation.md for the data behind this choice.
//
// The returned diffs are sorted by (repository, name) for deterministic output.
func ComputeDiff(installed, candidate []ImageCVEs) []ImageDiff {
	installedByKey := indexByPairKey(installed)
	candidateByKey := indexByPairKey(candidate)

	keys := unionKeys(installedByKey, candidateByKey)

	diffs := make([]ImageDiff, 0, len(keys))
	for _, key := range keys {
		inst, hasInst := installedByKey[key]
		cand, hasCand := candidateByKey[key]

		switch {
		case hasInst && hasCand:
			diffs = append(diffs, pairedDiff(inst, cand))
		case hasCand:
			// Image only in the candidate bundle: every CVE is newly introduced.
			diffs = append(diffs, ImageDiff{
				Repository:      cand.Repository,
				Name:            cand.Name,
				Status:          ImageAdded,
				CandidateDigest: cand.Digest,
				New:             sortedCVEs(cand.CVEs),
			})
		default:
			// Image only in the installed bundle: every CVE goes away with the upgrade.
			diffs = append(diffs, ImageDiff{
				Repository:      inst.Repository,
				Name:            inst.Name,
				Status:          ImageRemoved,
				InstalledDigest: inst.Digest,
				Fixed:           sortedCVEs(inst.CVEs),
			})
		}
	}
	return diffs
}

func pairedDiff(installed, candidate ImageCVEs) ImageDiff {
	installedCVEs := indexCVEs(installed.CVEs)
	candidateCVEs := indexCVEs(candidate.CVEs)

	diff := ImageDiff{
		Repository:      candidate.Repository,
		Name:            candidate.Name,
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

// pairKey is the diff pairing key for an image: (repository, name). The NUL separator cannot
// occur in either component, so distinct (repository, name) pairs never collide. Images that
// share a repository but differ in name (multi-version bundles) get distinct keys; images with
// an empty name are disambiguated by repository. Validated unique across sampled operators (see
// mt_nogit_image_remed/phase1_validation.md).
func pairKey(img ImageCVEs) string {
	return img.Repository + "\x00" + img.Name
}

// indexByPairKey indexes images by their (repository, name) key. If two images share a key the
// last one wins; that pairing is validated not to occur within real bundles.
func indexByPairKey(images []ImageCVEs) map[string]ImageCVEs {
	byKey := make(map[string]ImageCVEs, len(images))
	for _, img := range images {
		byKey[pairKey(img)] = img
	}
	return byKey
}

// indexCVEs indexes CVEs by ID. If a CVE ID appears more than once the last one wins.
func indexCVEs(cves []CVE) map[string]CVE {
	byID := make(map[string]CVE, len(cves))
	for _, c := range cves {
		byID[c.ID] = c
	}
	return byID
}

func unionKeys(a, b map[string]ImageCVEs) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	for key := range a {
		seen[key] = struct{}{}
	}
	for key := range b {
		seen[key] = struct{}{}
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	// Keys are "repository\x00name"; sorting the raw key orders by repository then name.
	sort.Strings(keys)
	return keys
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
