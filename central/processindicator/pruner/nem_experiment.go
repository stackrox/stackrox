package pruner

// EXPERIMENT: This file is throwaway instrumentation for PR #22014 validation.
// It runs the normalized-exact-match algorithm and tracks which anchor each
// indicator matched, for shadow-table recording.

import (
	"github.com/stackrox/rox/central/processindicator"
)

// NEMResult records the NEM algorithm's decision for a single indicator.
type NEMResult struct {
	ID              string
	WouldPrune      bool
	NormalizedArgs  string
	MatchedAnchorID string
}

// NEMPrune runs the normalized-exact-match algorithm on the given processes
// and returns a result for each indicator.
func NEMPrune(processes []processindicator.IDAndArgs, minProcesses int) []NEMResult {
	results := make([]NEMResult, 0, len(processes))

	if len(processes) <= minProcesses {
		for _, p := range processes {
			results = append(results, NEMResult{ID: p.ID, NormalizedArgs: normalizeArgs(p.Args)})
		}
		return results
	}

	seen := make(map[string]string) // normalized args → anchor ID
	pruneCount := 0

	for _, process := range processes {
		if len(processes)-pruneCount <= minProcesses {
			results = append(results, NEMResult{ID: process.ID, NormalizedArgs: normalizeArgs(process.Args)})
			continue
		}

		normalized := normalizeArgs(process.Args)

		if anchorID, exists := seen[normalized]; exists {
			results = append(results, NEMResult{
				ID:              process.ID,
				WouldPrune:      true,
				NormalizedArgs:  normalized,
				MatchedAnchorID: anchorID,
			})
			pruneCount++
		} else {
			seen[normalized] = process.ID
			results = append(results, NEMResult{
				ID:             process.ID,
				NormalizedArgs: normalized,
			})
		}
	}

	return results
}
