// conflicts.go warns when a run rule and a skip rule can match the same
// file. The crossing is legal. An inexact automaton still warns, without
// an example path.

package rulesvalidation

import (
	"fmt"
	"slices"
	"strings"

	"github.com/stackrox/rox/ci/select/internal/overlap"
	"github.com/stackrox/rox/ci/select/internal/rules"
)

// Check reports the run/skip conflicts in the mapped rules, one bare
// line per contested pair; report in full.go numbers the findings and
// appends the footer. Full.Validate runs it after the structural
// checks.
func Check(m rules.Mapping) []string {
	runRules, skipRules := partitionRules(m.Rules)
	var conflicts []string
	for _, run := range runRules {
		for _, skip := range skipRules {
			jobs := overlappingTargets(run.Run, skip.Skip)
			if len(jobs) == 0 {
				continue
			}
			meeting := overlap.Of(run, skip)
			if !meeting.Meets {
				continue
			}
			// Every file the run matches is skipped here. That is an
			// error, reported with the contradictions, so it is not also
			// a warning that a diff might miss.
			if meeting.Exact {
				cover := overlap.Covers(skip, run)
				if cover.Exact && cover.Covers {
					continue
				}
			}
			witness := trustedWitness(run, skip, meeting.Witness)
			witnessDetail := ""
			if witness != "" {
				witnessDetail = fmt.Sprintf(": both match e.g. %q", witness)
			}
			conflicts = append(conflicts, fmt.Sprintf(
				"%q vs %q for %s%s",
				run.Name, skip.Name, joinQuoted(jobs), witnessDetail))
		}
	}
	if len(conflicts) == 0 {
		return nil
	}
	slices.Sort(conflicts)
	return conflicts
}

func partitionRules(all []*rules.Rule) (runRules, skipRules []*rules.Rule) {
	for _, r := range all {
		// The final remaining rule is excluded by its `when`, not by its
		// run list: a path rule with run: ['*'] joins the run side like
		// any other and conflicts with every skip covering its files.
		if r.When != rules.WhenAnyFileMatches {
			continue
		}
		if len(r.Run) > 0 {
			runRules = append(runRules, r)
		}
		if len(r.Skip) > 0 {
			skipRules = append(skipRules, r)
		}
	}
	return runRules, skipRules
}

// overlappingTargets returns the sorted jobs the run rule requests that the
// skip rule covers, considering the '*' wildcard.
func overlappingTargets(run, skip []string) []string {
	var out []string
	if slices.Contains(run, "*") {
		out = append(out, skip...)
	} else {
		for _, j := range run {
			if slices.Contains(skip, j) {
				out = append(out, j)
			}
		}
	}
	slices.Sort(out)
	return out
}

func joinQuoted(items []string) string {
	quoted := make([]string, len(items))
	for i, item := range items {
		quoted[i] = fmt.Sprintf("%q", item)
	}
	return strings.Join(quoted, ", ")
}
