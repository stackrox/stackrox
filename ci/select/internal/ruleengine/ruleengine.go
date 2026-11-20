// Package ruleengine is the seam between the selection callers (the CLI
// and, later, the GitHub Actions planner) and a rules engine. Callers
// depend only on this package, so replacing the engine means adding an
// implementation.
package ruleengine

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// Decision is one selection run's plan: an action ("run" or "skip") per
// CI target, plus a human-readable trace.
type Decision struct {
	// Actions maps a target to "run" or "skip". Undecided targets are
	// absent; Action reports "run" for them, so callers fail safe.
	Actions map[string]string
	// Log explains the plan for humans.
	Log string
}

// Action reports the action for one target; anything undecided runs.
func (d Decision) Action(target string) string {
	if action := d.Actions[target]; action != "" {
		return action
	}
	return "run"
}

func (d Decision) String() string {
	if len(d.Actions) == 0 {
		return "all targets run\n"
	}
	var text strings.Builder
	for _, job := range slices.Sorted(maps.Keys(d.Actions)) {
		fmt.Fprintf(&text, "%s %s\n", job, d.Actions[job])
	}
	return text.String()
}

// Engine turns a rules document and the changed paths into a plan. The
// engine owns the rules format, so swapping engines swaps the language with
// it. changed is the pull request's changed files, detected upstream (for
// example by the detect-changes workflow) and passed in.
type Engine interface {
	Plan(rules []byte, changed []string) (Decision, error)
}
