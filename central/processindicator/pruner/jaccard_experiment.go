package pruner

// EXPERIMENT: This file is throwaway instrumentation for PR #22014 validation.
// It contains the original Jaccard similarity algorithm (verbatim from the
// pre-#22014 code, using RoaringBitmap) extended to return per-indicator
// results with similarity scores and anchor IDs for shadow-table recording.

import (
	"strings"

	"github.com/RoaringBitmap/roaring/v2"
	"github.com/stackrox/rox/central/processindicator"
)

const jaccardThreshold = 0.6

// JaccardResult records the Jaccard algorithm's decision for a single indicator.
type JaccardResult struct {
	ID              string
	WouldPrune      bool
	Similarity      float64
	MatchedAnchorID string
}

// JaccardPrune runs the original Jaccard similarity algorithm on the given
// processes and returns a result for each indicator indicating whether it
// would be pruned, along with the similarity score and matched anchor.
func JaccardPrune(processes []processindicator.IDAndArgs, minProcesses int) []JaccardResult {
	results := make([]JaccardResult, 0, len(processes))

	if len(processes) <= minProcesses {
		for _, p := range processes {
			results = append(results, JaccardResult{ID: p.ID})
		}
		return results
	}

	knownStrings := make(map[string]int)

	type anchorEntry struct {
		id     string
		bitmap *roaring.Bitmap
	}
	anchors := make([]anchorEntry, 0, minProcesses)
	pruneCount := 0

	for _, process := range processes {
		if len(processes)-pruneCount <= minProcesses {
			results = append(results, JaccardResult{ID: process.ID})
			continue
		}

		normalized := jaccardNormalizeArgs(process.Args, knownStrings)

		bestSim := 0.0
		bestAnchorID := ""
		for _, anchor := range anchors {
			sim := jaccardSimilarityScore(anchor.bitmap, normalized)
			if sim > bestSim {
				bestSim = sim
				bestAnchorID = anchor.id
			}
		}

		if bestSim >= jaccardThreshold {
			results = append(results, JaccardResult{
				ID:              process.ID,
				WouldPrune:      true,
				Similarity:      bestSim,
				MatchedAnchorID: bestAnchorID,
			})
			pruneCount++
		} else {
			anchors = append(anchors, anchorEntry{id: process.ID, bitmap: normalized})
			results = append(results, JaccardResult{ID: process.ID})
		}
	}

	return results
}

// jaccardNormalizeArgs is the original normalizeArgs function from the
// pre-#22014 code, using RoaringBitmap.
func jaccardNormalizeArgs(args string, knownStrings map[string]int) *roaring.Bitmap {
	words := strings.Fields(args)
	bitmap := roaring.New()
	for _, word := range words {
		normalized := numericRegex.ReplaceAllString(word, "#")
		var val int
		if mapValue, ok := knownStrings[normalized]; ok {
			val = mapValue
		} else {
			val = len(knownStrings)
			knownStrings[normalized] = val
		}
		bitmap.AddInt(val)
	}
	return bitmap
}

func jaccardSimilarityScore(first, second *roaring.Bitmap) float64 {
	return float64(first.AndCardinality(second)) / float64(first.OrCardinality(second))
}
