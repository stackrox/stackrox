package resolver

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/ci/select/internal/rules"
)

// loggedPathLimit caps the path list so a large diff cannot flood the log.
const loggedPathLimit = 50

const traceLegend = "Trace legend: every job is governed by exactly one rule, or by always_run.\n" +
	"A rule with run_if_changed runs its jobs when any changed file matches,\n" +
	"else skips them. A rule with skip_if_only_changed skips its jobs when every\n" +
	"changed file matches, else runs them. A rule with both runs on a positive\n" +
	"match or on any file the skip set does not cover; on an overlap, run wins.\n" +
	"An empty diff runs every job.\n\n"

func renderTrace(files []string, m rules.Mapping, outcomes []ruleOutcome) string {
	var b strings.Builder
	b.WriteString(traceLegend)

	b.WriteString(renderPaths(files))

	if len(m.AlwaysRun) > 0 {
		fmt.Fprintf(&b, "\nalways_run: %s\n", strings.Join(m.AlwaysRun, ", "))
	}

	b.WriteString("\nRules:\n")
	for _, outcome := range outcomes {
		b.WriteString(renderRule(outcome))
	}
	return b.String()
}

func renderPaths(files []string) string {
	if len(files) == 0 {
		return "Changed paths: none (empty diff); every job runs.\n"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Changed paths (%d):\n", len(files))
	for i, path := range files {
		if i == loggedPathLimit {
			fmt.Fprintf(&b, "  ... and %d more\n", len(files)-loggedPathLimit)
			break
		}
		fmt.Fprintf(&b, "  %d. %s\n", i+1, path)
	}
	return b.String()
}

func renderRule(o ruleOutcome) string {
	return fmt.Sprintf("- %q -> %s %s\n  (%s)\n",
		o.name, o.action, strings.Join(o.jobs, ", "), ruleReason(o))
}

func ruleReason(o ruleOutcome) string {
	switch {
	case o.hasRun && !o.hasSkip:
		if o.runMatch {
			return "a changed file matches run_if_changed"
		}
		return "no changed file matches run_if_changed"
	case !o.hasRun && o.hasSkip:
		if o.skipCover {
			return "every changed file matches skip_if_only_changed"
		}
		return "a changed file is outside skip_if_only_changed"
	case o.hasRun && o.hasSkip:
		switch {
		case o.runMatch && o.skipCover:
			return "run_if_changed matches and skip_if_only_changed covers the diff; run wins"
		case o.runMatch:
			return "a changed file matches run_if_changed"
		case !o.skipCover:
			return "a changed file is outside skip_if_only_changed"
		default:
			return "every changed file matches skip_if_only_changed and nothing matches run_if_changed"
		}
	default:
		return "the rule declares no path conditions"
	}
}
