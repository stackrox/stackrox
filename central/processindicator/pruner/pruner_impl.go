package pruner

import (
	"strings"
	"time"

	"github.com/stackrox/rox/central/processindicator"
)

type prunerFactoryImpl struct {
	minProcesses int
	period       time.Duration
}

func normalizeArgs(args string) string {
	words := strings.Fields(args)
	normalized := make([]string, 0, len(words))
	for _, word := range words {
		normalized = append(normalized, numericRegex.ReplaceAllString(word, "#"))
	}
	return strings.Join(normalized, " ")
}

func (p *prunerFactoryImpl) Prune(processes []processindicator.IDAndArgs) (idsToRemove []string) {
	if len(processes) <= p.minProcesses {
		return nil
	}

	seen := make(map[string]struct{})

	for _, process := range processes {
		if len(processes)-len(idsToRemove) <= p.minProcesses {
			return
		}
		normalized := normalizeArgs(process.Args)
		if _, exists := seen[normalized]; !exists {
			seen[normalized] = struct{}{}
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
