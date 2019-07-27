package pruner

import (
	"regexp"
	"time"

	"github.com/stackrox/rox/central/processindicator"
)

var (
	numericRegex = regexp.MustCompile(`\d+`)
)

// A Pruner prunes process indicators.
type Pruner interface {
	// Prune takes the given args and returns the ids that can be pruned.
	Prune([]processindicator.IDAndArgs) (idsToRemove []string)
	// Finish indicates that the current pruner is done being used.
	// (The current prunerImpl is stateless, but this is helpful in unit tests.)
	Finish()
}

// A Factory allows creating pruners for periodic pruning.
// Each pruning run is initiated by calling `StartPruning()` and then calling `Prune()` repeatedly on the returned
// `Pruner`.
type Factory interface {
	StartPruning() Pruner
	Period() time.Duration
}

//go:generate mockgen-wrapper
