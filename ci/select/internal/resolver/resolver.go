// Package resolver decides run or skip for each CI job. Inputs must have
// passed validation.Load. Because each job belongs to exactly one rule, a
// rule decides its whole group at once; see decideRule for the semantics.
package resolver

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/stackrox/rox/ci/select/internal/rules"
)

// Decision carries one action ("run" or "skip") per CI job plus the trace.
type Decision struct {
	Actions map[string]string
	Log     string
}

// ruleOutcome is the evidence trace.go needs to explain one rule's verdict.
type ruleOutcome struct {
	name      string
	jobs      []string
	hasRun    bool
	hasSkip   bool
	runMatch  bool
	skipCover bool
	action    string
}

func Resolve(files []string, m rules.Mapping) Decision {
	actions := make(map[string]string, len(m.Jobs))
	for _, job := range m.AlwaysRun {
		actions[job] = "run"
	}

	outcomes := make([]ruleOutcome, 0, len(m.Rules))
	for _, r := range m.Rules {
		outcome := decideRule(files, r)
		outcomes = append(outcomes, outcome)
		for _, job := range r.Jobs {
			actions[job] = outcome.action
		}
	}

	return Decision{Actions: actions, Log: renderTrace(files, m, outcomes)}
}

// decideRule computes the verdict for one rule's jobs. The clause the rule
// declares sets the default: run_if_changed defaults to skip (run only on a
// match), skip_if_only_changed defaults to run (skip only when it covers the
// whole diff). With both, run wins — the safe bias when the two disagree.
// An empty diff always runs: nothing changed that a rule could clear.
func decideRule(files []string, r *rules.Rule) ruleOutcome {
	outcome := ruleOutcome{
		name:      r.Name,
		jobs:      r.Jobs,
		hasRun:    r.HasRun(),
		hasSkip:   r.HasSkip(),
		runMatch:  r.RunMatchesAny(files),
		skipCover: r.SkipCoversAll(files),
	}

	run := true
	switch {
	case len(files) == 0:
		run = true
	case outcome.hasRun && !outcome.hasSkip:
		run = outcome.runMatch
	case !outcome.hasRun && outcome.hasSkip:
		run = !outcome.skipCover
	case outcome.hasRun && outcome.hasSkip:
		run = outcome.runMatch || !outcome.skipCover
	}

	outcome.action = "skip"
	if run {
		outcome.action = "run"
	}
	return outcome
}

// Fallback is the fail-open decision: nil Actions, so Action returns "run"
// for every job.
func Fallback(err error) Decision {
	return Decision{Log: fmt.Sprintf("selection unavailable: %v; all jobs run\n", err)}
}

// Action defaults to "run" for any job with no recorded decision (fail-open).
func (d Decision) Action(job string) string {
	if action := d.Actions[job]; action != "" {
		return action
	}
	return "run"
}

func (d Decision) String() string {
	if d.Actions == nil {
		return "all jobs run\n"
	}
	var text strings.Builder
	for _, job := range slices.Sorted(maps.Keys(d.Actions)) {
		fmt.Fprintf(&text, "%s %s\n", job, d.Actions[job])
	}
	return text.String()
}
