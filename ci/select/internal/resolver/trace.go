package resolver

import (
	"fmt"
	"strings"
)

// loggedPathLimit caps the path list so a large diff cannot flood the log.
const loggedPathLimit = 50

const traceLegend = "Trace legend: every job is governed by exactly one rule.\n" +
	"A run_if_changed rule runs its jobs when any changed file matches, else\n" +
	"skips them. A skip_if_only_changed rule skips its jobs when every changed\n" +
	"file matches, else runs them. An always_run rule always runs its jobs.\n" +
	"An empty diff runs every job.\n\n"

func renderTrace(files []string, outcomes []ruleOutcome) string {
	var b strings.Builder
	b.WriteString(traceLegend)

	b.WriteString(renderPaths(files))

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
	case o.alwaysRun:
		return "always_run"
	case o.hasRun:
		if o.runMatch {
			return "a changed file matches run_if_changed"
		}
		return "no changed file matches run_if_changed"
	default:
		if o.skipCover {
			return "every changed file matches skip_if_only_changed"
		}
		return "a changed file is outside skip_if_only_changed"
	}
}
