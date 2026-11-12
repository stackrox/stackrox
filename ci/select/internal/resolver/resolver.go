// Package resolver decides run or skip for each CI job from the changed paths.
// Inputs must have passed validation.
package resolver

import (
	"fmt"

	"github.com/stackrox/rox/ci/select/internal/ruleengine"
	"github.com/stackrox/rox/ci/select/internal/rules"
)

// ruleOutcome is the evidence trace.go needs to explain one rule's verdict.
type ruleOutcome struct {
	name      string
	jobs      []string
	alwaysRun bool
	hasRun    bool
	runMatch  bool
	skipCover bool
	action    string
}

// Resolve decides run or skip for every job in the mapping from the changed files.
func Resolve(files []string, m rules.Mapping) ruleengine.Decision {
	actions := make(map[string]string, len(m.Jobs))
	outcomes := make([]ruleOutcome, 0, len(m.Rules))
	for _, r := range m.Rules {
		outcome := decideRule(files, r)
		outcomes = append(outcomes, outcome)
		for _, job := range r.Jobs {
			actions[job] = outcome.action
		}
	}
	return ruleengine.Decision{Actions: actions, Log: renderTrace(files, outcomes)}
}

func decideRule(files []string, r *rules.Rule) ruleOutcome {
	outcome := ruleOutcome{
		name:      r.Name,
		jobs:      r.Jobs,
		alwaysRun: r.AlwaysRun,
		hasRun:    r.HasRun(),
		runMatch:  r.RunMatchesAny(files),
		skipCover: r.SkipCoversAll(files),
	}

	run := true
	switch {
	case r.AlwaysRun:
		run = true
	case len(files) == 0: // empty diff: nothing a skip rule could clear
		run = true
	case r.HasRun():
		run = outcome.runMatch
	case r.HasSkip():
		run = !outcome.skipCover
	}

	outcome.action = "skip"
	if run {
		outcome.action = "run"
	}
	return outcome
}

// Fallback fails open: nil Actions makes Decision.Action return "run" for every job.
func Fallback(err error) ruleengine.Decision {
	return ruleengine.Decision{Log: fmt.Sprintf("selection unavailable: %v; all jobs run\n", err)}
}
