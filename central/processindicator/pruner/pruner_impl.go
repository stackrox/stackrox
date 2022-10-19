package pruner

import (
	"context"
	"strings"
	"time"

	"github.com/RoaringBitmap/roaring"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/processindicator"
	"github.com/stackrox/rox/pkg/logging"
)

const (
	jaccardThreshold = 0.6

	pruneOrphanedPodIndicatorsStmt = `DELETE FROM process_indicators child WHERE NOT EXISTS
		(SELECT 1 FROM pods parent WHERE child.poduid = parent.Id) and child.signal_time < $1`

	pruneOrphanedDeploymentIndicatorsStmt = `DELETE FROM process_indicators child WHERE NOT EXISTS
		(SELECT 1 FROM deployments parent WHERE child.deploymentid = parent.Id) and child.signal_time < $1`
)

var (
	log = logging.LoggerForModule()
)

type prunerFactoryImpl struct {
	minProcesses int
	period       time.Duration
}

func normalizeWord(word string) string {
	return numericRegex.ReplaceAllString(word, "#")
}

// knownStrings maps each string we see to a unique integer.
func normalizeArgs(args string, knownStrings map[string]int) *roaring.Bitmap {
	words := strings.Fields(args)

	bitmap := roaring.New()
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
		bitmap.AddInt(val)
	}
	return bitmap
}

func jaccardSimilarity(first, second *roaring.Bitmap) float64 {
	return float64(first.AndCardinality(second)) / float64(first.OrCardinality(second))
}

func isCloseToAnExistingSet(existingSets []*roaring.Bitmap, candidate *roaring.Bitmap) bool {
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

	prunedNormalized := make([]*roaring.Bitmap, 0, p.minProcesses)

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

// PruneOrphanedPodIndicators - prunes process indicators whose pod no longer exists
func PruneOrphanedPodIndicators(ctx context.Context, pool *pgxpool.Pool, orphanedBefore time.Time) {
	if _, err := pool.Exec(ctx, pruneOrphanedPodIndicatorsStmt, orphanedBefore); err != nil {
		log.Errorf("failed to prune orhpaned pod indicators: %v", err)
	}
}

// PruneOrphanedDeploymentIndicators - prunes process indicators whose deployment no longer exists
func PruneOrphanedDeploymentIndicators(ctx context.Context, pool *pgxpool.Pool, orphanedBefore time.Time) {
	if _, err := pool.Exec(ctx, pruneOrphanedDeploymentIndicatorsStmt, orphanedBefore); err != nil {
		log.Errorf("failed to prune orhpaned pod indicators: %v", err)
	}
}

// NewFactory returns a new Factory that creates pruners never pruning below the given number of `minProcesses`.
func NewFactory(minProcesses int, period time.Duration) Factory {
	return &prunerFactoryImpl{
		minProcesses: minProcesses,
		period:       period,
	}
}
