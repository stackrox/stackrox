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

type prunerFactoryImpl struct {
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

func (p *prunerFactoryImpl) Prune(processes []processindicator.IDAndArgs) (idsToRemove []string) {
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

func (p *prunerFactoryImpl) Finish() {}

func (p *prunerFactoryImpl) Period() time.Duration {
	return p.period
}

func (p *prunerFactoryImpl) StartPruning() Pruner {
	return p
}

// NewFactory returns an new Factory that creates pruners never pruning below the given number of `minProcesses`.
func NewFactory(minProcesses int, period time.Duration) Factory {
	return &prunerFactoryImpl{
		minProcesses: minProcesses,
		period:       period,
	}
}
