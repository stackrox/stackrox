package resolver

// trace.go turns decisionTrace into the numbered log. Those sentences
// are built only from the evidence the decision pass recorded.

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/stackrox/rox/ci/select/internal/rules"
)

// loggedPathLimit bounds how many changed paths the explanation lists.
// pathRefLimit bounds the path-number citations in per-job explanations;
// numbers are compact, so more of them fit before folding to a count.
const (
	loggedPathLimit = 50
	pathRefLimit    = 20
)

// traceLegend heads every trace and defines the vocabulary the numbered
// sections below it use.
const traceLegend = "Trace legend: rules either request runs for a job or mark paths safe to skip\n" +
	"it. A run request counts only for a path not marked safe to skip. Per job:\n" +
	"a counting run request runs it; otherwise full skip coverage skips it;\n" +
	"otherwise \"remaining\" runs it. On a conflict for one path, the skip wins:\n" +
	"it marks the file safe to skip, so the run request does not count there.\n" +
	"Conflicts are reported with the trace; validation also reports them as\n" +
	"warnings when the rules file is checked. Changed paths and decisions are\n" +
	"numbered; decisions cite path numbers.\n\n"

// describe renders one defeated run request: who requested, and which
// covering skip rules won the path.
func (e crossEvent) describe(claim crossClaim) string {
	return fmt.Sprintf("- conflict on path %d (%s): %q requests run, %s marks it safe to skip, so the skip wins",
		e.pathNumber, e.path, claim.runRule, quoteList(claim.coverRules))
}

// explain builds the sentences for one job's decision from its evidence:
// the decided action, which rules requested or covered, and the conflicts
// crossed. The remaining rule fills in the fallback sentences from the
// name the decision pass recorded for it. Jobs no rule decided get no
// lines, matching the decision pass leaving them undecided.
func explain(files []string, t *decisionTrace) []string {
	var lines []string
	switch {
	case t.action == "skip":
		lines = append(lines,
			"skip, because every changed path is covered and no run request survived",
			fmt.Sprintf("- covered by: %s", quoteList(slices.Sorted(maps.Keys(t.coverNames)))))
	case len(t.runNames) > 0:
		lines = append(lines,
			"run, because a run request survived",
			fmt.Sprintf("- run requested by %s for paths: %s",
				quoteList(slices.Sorted(maps.Keys(t.runNames))), describePathRefs(t.runPaths)))
	case t.remaining != "":
		if len(files) == 0 {
			lines = append(lines,
				fmt.Sprintf("run, because an empty diff provides no coverage, so %q runs the job", t.remaining))
		} else {
			lines = append(lines,
				fmt.Sprintf("run, because not every path is covered, so %q runs the job", t.remaining),
				fmt.Sprintf("- uncovered paths: %s", describePathRefs(t.uncovered)))
		}
	}
	for _, cross := range t.crosses {
		for _, claim := range cross.claims {
			lines = append(lines, cross.describe(claim))
		}
	}
	return lines
}

// renderTrace writes the full explanation: the numbered changed paths,
// then the numbered per-job decisions, with identical explanations
// sharing one grouped line. Job explanations cite changed-path numbers
// (1-based, diff order), so the per-file section stays the single place
// listing full paths.
func renderTrace(files []string, m rules.Mapping, matches []fileMatches, traces map[string]*decisionTrace) string {
	var log strings.Builder
	log.WriteString(traceLegend)
	log.WriteString("-- changed paths --\n")
	// Cap the per-path lines: huge diffs otherwise overflow logs and the
	// 1 MB step summary while the per-job reasons below stay complete.
	// Numbered lines can be cited in reviews and CI discussions.
	for i, match := range matches {
		if i == loggedPathLimit {
			fmt.Fprintf(&log, "changed paths: %d more not listed\n", len(files)-i)
			break
		}
		// Sorted names keep the explanation independent of rule order.
		names := make([]string, 0, len(match.rules))
		for _, r := range match.rules {
			names = append(names, r.Name)
		}
		slices.Sort(names)
		if len(names) == 0 {
			fmt.Fprintf(&log, "%d. changed path: %q matched: none\n", i+1, match.path)
		} else {
			fmt.Fprintf(&log, "%d. changed path: %q matched: %s\n", i+1, match.path, strings.Join(names, ", "))
		}
	}
	if len(files) == 0 {
		log.WriteString("empty diff: no skip coverage\n")
	}

	log.WriteString("\n-- decisions per job --\n")
	type group struct {
		jobs  []string
		lines []string
	}
	var groups []*group
	byExplanation := make(map[string]*group)
	for _, job := range m.Jobs {
		lines := explain(files, traces[job])
		key := strings.Join(lines, "\n")
		g, ok := byExplanation[key]
		if !ok {
			g = &group{lines: lines}
			byExplanation[key] = g
			groups = append(groups, g)
		}
		g.jobs = append(g.jobs, job)
	}
	lineNumber := 0
	for _, g := range groups {
		for i, line := range g.lines {
			if i == 0 {
				lineNumber++
				fmt.Fprintf(&log, "%d. %s: %s\n", lineNumber, describeJobs(g.jobs), line)
			} else {
				fmt.Fprintf(&log, "   %s\n", line)
			}
		}
	}
	return log.String()
}

// describeJobs lists job names, folding long lists to a count so grouped
// explanation lines stay readable.
func describeJobs(jobs []string) string {
	if len(jobs) > 4 {
		return fmt.Sprintf("%s (+%d more)", strings.Join(jobs[:3], ", "), len(jobs)-3)
	}
	return strings.Join(jobs, ", ")
}

// quoteList joins rule names so a CI log reads as words, not a Go slice.
func quoteList(names []string) string {
	quoted := make([]string, len(names))
	for i, name := range names {
		quoted[i] = strconv.Quote(name)
	}
	return strings.Join(quoted, ", ")
}

// describePathRefs renders changed-path numbers for the explanation, folding
// long lists to a count like the changed-path section.
func describePathRefs(numbers []int) string {
	if len(numbers) == 0 {
		return "none"
	}
	text := make([]string, 0, len(numbers))
	for _, n := range numbers {
		text = append(text, strconv.Itoa(n))
	}
	if len(text) > pathRefLimit {
		return fmt.Sprintf("%s (+%d more)", strings.Join(text[:pathRefLimit], ", "), len(text)-pathRefLimit)
	}
	return strings.Join(text, ", ")
}
