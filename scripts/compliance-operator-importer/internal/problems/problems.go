// Package problems provides a Collector that accumulates Problem entries
// during an importer run. All collected problems are included in the final
// JSON report and used to determine the process exit code.
package problems

import "github.com/stackrox/co-acs-importer/internal/models"

// Collector accumulates problems during a run.
// It is not safe for concurrent use; callers must synchronise externally if needed.
type Collector struct {
	problems []models.Problem
}

// NewCollector returns an empty Collector ready for use.
func NewCollector() *Collector {
	return &Collector{}
}

// Add appends p to the collected problem list.
// Both Description and FixHint must be non-empty to satisfy IMP-CLI-022.
func (c *Collector) Add(p models.Problem) {
	c.problems = append(c.problems, p)
}

// All returns a copy of all collected problems in insertion order.
func (c *Collector) All() []models.Problem {
	if len(c.problems) == 0 {
		return []models.Problem{}
	}
	out := make([]models.Problem, len(c.problems))
	copy(out, c.problems)
	return out
}

// HasErrors returns true if at least one collected problem has severity "error".
// Used to determine whether the run should exit with code 2 (IMP-CLI-019).
func (c *Collector) HasErrors() bool {
	for _, p := range c.problems {
		if p.Severity == models.SeverityError {
			return true
		}
	}
	return false
}
