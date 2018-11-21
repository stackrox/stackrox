package pruner

import (
	"strings"
	"time"

	"github.com/stackrox/rox/central/processindicator"
	"github.com/stackrox/rox/pkg/set"
)

const (
	jaccardThreshold = 0.6
)

type prunerImpl struct {
	minProcesses int
	period       time.Duration
}

func normalizeWord(word string) string {
	return numericRegex.ReplaceAllString(word, "#")
}

// knownStrings maps each string we see to a unique integer.
func normalizeArgs(args string, knownStrings map[string]int) set.IntSet {
	words := strings.Fields(args)

	intSet := set.NewIntSet()
	for _, word := range words {
		normalized := normalizeWord(word)
		var val int
		if mapValue, ok := knownStrings[normalized]; ok {
			val = mapValue
		} else {
			// If this is a previously unseen string, assign it the next available integer (for which
			// we just use the current length of the map), and add it to knownStrings.
			val = len(knownStrings)
			knownStrings[normalized] = val
		}
		intSet.Add(val)
	}
	return intSet
}

func jaccardSimilarity(first, second set.IntSet) float64 {
	return float64(first.Intersect(second).Cardinality()) / float64(first.Union(second).Cardinality())
}

func isCloseToAnExistingSet(existingSets []set.IntSet, candidate set.IntSet) bool {
	for _, existingSet := range existingSets {
		if jaccardSimilarity(existingSet, candidate) >= jaccardThreshold {
			return true
		}
	}
	return false
}

func (p *prunerImpl) Prune(processes []processindicator.IDAndArgs) (idsToRemove []string) {
	knownStrings := make(map[string]int)

	if len(processes) <= p.minProcesses {
		return nil
	}

	prunedNormalized := make([]set.IntSet, 0, p.minProcesses)

	for _, process := range processes {
		if len(processes)-len(idsToRemove) <= p.minProcesses {
			return
		}
		normalized := normalizeArgs(process.Args, knownStrings)
		if !isCloseToAnExistingSet(prunedNormalized, normalized) {
			prunedNormalized = append(prunedNormalized, normalized)
		} else {
			idsToRemove = append(idsToRemove, process.ID)
		}
	}

	return
}

func (p *prunerImpl) Period() time.Duration {
	return p.period
}

// New returns an new Pruner that never prunes below the given number of `minProcesses`.
func New(minProcesses int, period time.Duration) Pruner {
	return &prunerImpl{
		minProcesses: minProcesses,
		period:       period,
	}
}
