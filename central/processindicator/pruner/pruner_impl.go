package pruner

import (
	"strings"
	"time"

	"github.com/stackrox/rox/central/processindicator"
)

const (
	jaccardThreshold = 0.6
)

type wordSet map[string]struct{}

type prunerFactoryImpl struct {
	minProcesses int
	period       time.Duration
}

func normalizeWord(word string) string {
	return numericRegex.ReplaceAllString(word, "#")
}

func normalizeArgs(args string) wordSet {
	words := strings.Fields(args)
	set := make(wordSet, len(words))
	for _, word := range words {
		set[normalizeWord(word)] = struct{}{}
	}
	return set
}

func jaccardSimilarity(a, b wordSet) float64 {
	// Iterate over the smaller set for efficiency.
	if len(a) > len(b) {
		a, b = b, a
	}
	var intersection int
	for w := range a {
		if _, ok := b[w]; ok {
			intersection++
		}
	}
	union := len(a) + len(b) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

func isCloseToAnExistingSet(existingSets []wordSet, candidate wordSet) bool {
	for _, existingSet := range existingSets {
		if jaccardSimilarity(existingSet, candidate) >= jaccardThreshold {
			return true
		}
	}
	return false
}

func (p *prunerFactoryImpl) Prune(processes []processindicator.IDAndArgs) (idsToRemove []string) {
	if len(processes) <= p.minProcesses {
		return nil
	}

	prunedNormalized := make([]wordSet, 0, p.minProcesses)

	for _, process := range processes {
		if len(processes)-len(idsToRemove) <= p.minProcesses {
			return
		}
		normalized := normalizeArgs(process.Args)
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
