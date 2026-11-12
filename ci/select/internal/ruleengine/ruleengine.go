// Package ruleengine defines the interface used to match the changed paths
// against rules that defines what jobs to run or skip
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

// String renders the plan as sorted "<target> <action>" lines.
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

// Engine selects jobs for one rules format; replacing it replaces the language.
type Engine interface {
	// Plan matches changed against rules to decide run/skip per job. changed is
	// the pull request's changed files, detected upstream (e.g. detect-changes);
	// rules is the rules document.
	Plan(rules []byte, changed []string) (Decision, error)
}
