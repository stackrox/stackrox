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
//go:generate mockgen-wrapper Pruner
type Pruner interface {
	// Prune takes the given args and returns the ids that can be pruned.
	Prune([]processindicator.IDAndArgs) (idsToRemove []string)

	Period() time.Duration
}
