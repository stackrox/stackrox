package main

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/stackrox/rox/pkg/set"
)

// loggedPathLimit bounds how many changed paths the explanation lists.
// pathRefLimit bounds the path-number citations in per-job explanations;
// numbers are compact, so more of them fit before folding to a count.
const (
	loggedPathLimit = 50
	pathRefLimit    = 20
)

type decision struct {
	actions map[string]string
	log     string
}

func resolve(files []string, m mapping) decision {
	actions := make(map[string]string, len(m.Jobs))
	var log strings.Builder
	log.WriteString("Trace legend: rules either request runs for a job or mark paths safe to skip\n" +
		"it. A run request counts only for a path not marked safe to skip. Per job:\n" +
		"a surviving run request runs it; otherwise full skip coverage skips it;\n" +
		"otherwise \"remaining\" runs it. On a conflict for one path, the skip wins\n" +
		"because it is specific to that file. Changed paths and decisions are\n" +
		"numbered; decisions cite path numbers.\n\n")
	log.WriteString("-- changed paths --\n")
	// Rules match files, not jobs: compute the match once per changed path.
	matches := make([]fileMatches, 0, len(files))
	for _, path := range files {
		match := fileMatches{path: path}
		for i := range m.Rules {
			r := &m.Rules[i]
			if r.When == "any-file-matches" && r.matches(path) {
				match.rules = append(match.rules, r)
			}
		}
		matches = append(matches, match)
	}
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
	// A job runs when a run request survives, skips when the diff is
	// nonempty and every changed path is covered, and otherwise falls
	// through to remaining with the paths that lacked coverage. Job
	// explanations cite changed-path numbers (1-based, diff order) so the
	// per-file section above stays the single place listing full paths.
	type crossEvent struct {
		pathNumber int
		path       string
		runRules   []string
		coverRules []string
	}
	type jobTrace struct {
		covered    bool
		coverNames set.StringSet
		runNames   set.StringSet
		runPaths   []int
		uncovered  []int
		crosses    []crossEvent
	}
	traces := make(map[string]*jobTrace, len(m.Jobs))
	explanations := make(map[string][]string, len(m.Jobs))
	for _, job := range m.Jobs {
		trace := &jobTrace{covered: len(files) > 0}
		traces[job] = trace
		for i, match := range matches {
			pathCovered := false
			var pathRuns, pathCovers []string
			for _, r := range match.rules {
				if r.requestsRun(job) {
					pathRuns = append(pathRuns, r.Name)
				}
				if r.skipsJob(job) {
					pathCovered = true
					pathCovers = append(pathCovers, r.Name)
					trace.coverNames.Add(r.Name)
				}
			}
			// A run request counts only for a file that this job's skip
			// rules do not already cover. One surviving request anywhere
			// in the diff still forces the job to run.
			if pathCovered {
				if len(pathRuns) > 0 {
					trace.crosses = append(trace.crosses, crossEvent{
						pathNumber: i + 1, path: match.path, runRules: pathRuns, coverRules: pathCovers,
					})
				}
			} else {
				for _, name := range pathRuns {
					trace.runNames.Add(name)
				}
				if len(pathRuns) > 0 {
					trace.runPaths = append(trace.runPaths, i+1)
				}
				trace.uncovered = append(trace.uncovered, i+1)
			}
			trace.covered = trace.covered && pathCovered
		}
		if len(trace.runNames) > 0 {
			actions[job] = "run"
			explanations[job] = append(explanations[job],
				"run, because a run request survived for a path not marked safe to skip",
				fmt.Sprintf("- run requested by %q for paths: %s", slices.Sorted(maps.Keys(trace.runNames)), describePathRefs(trace.runPaths)))
		} else if trace.covered {
			actions[job] = "skip"
			explanations[job] = append(explanations[job],
				"skip, because every changed path is covered and no run request survived",
				fmt.Sprintf("- covered by: %q", slices.Sorted(maps.Keys(trace.coverNames))))
		}
		for _, cross := range trace.crosses {
			explanations[job] = append(explanations[job],
				fmt.Sprintf("- conflict on path %d (%s): %q requests run, %q marks it safe to skip; the skip wins because it is specific to this file",
					cross.pathNumber, cross.path, cross.runRules, cross.coverRules))
		}
	}
	for _, r := range m.Rules {
		if r.When != "remaining" {
			continue
		}
		for _, job := range m.Jobs {
			if actions[job] != "" {
				continue
			}
			actions[job] = "run"
			if len(files) == 0 {
				explanations[job] = append(explanations[job],
					fmt.Sprintf("run, because an empty diff provides no coverage, so %q runs the job", r.Name))
				continue
			}
			explanations[job] = append(explanations[job],
				fmt.Sprintf("run, because not every path is covered, so %q runs the job", r.Name),
				fmt.Sprintf("- uncovered paths: %s", describePathRefs(traces[job].uncovered)))
		}
	}
	// Jobs with identical explanations share one grouped line each, so a
	// shared reason is not repeated for every target that carries it.
	// Numbered lines give every explanation a citable reference.
	log.WriteString("\n-- decisions per job --\n")
	type group struct {
		jobs  []string
		lines []string
	}
	var groups []*group
	byExplanation := make(map[string]*group)
	for _, job := range m.Jobs {
		key := strings.Join(explanations[job], "\n")
		g, ok := byExplanation[key]
		if !ok {
			g = &group{lines: explanations[job]}
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
	return decision{actions: actions, log: log.String()}
}

// describeJobs lists job names, folding long lists to a count so grouped
// explanation lines stay readable.
func describeJobs(jobs []string) string {
	if len(jobs) > 4 {
		return fmt.Sprintf("%s (+%d more)", strings.Join(jobs[:3], ", "), len(jobs)-3)
	}
	return strings.Join(jobs, ", ")
}

// fileMatches caches the rules matching one changed path. Rules match
// files, not jobs, so the regex work is shared by every job.
type fileMatches struct {
	path  string
	rules []*rule
}

// requestsRun reports whether the rule asks to run the job on any matched
// file, including through the '*' wildcard.
func (r *rule) requestsRun(job string) bool {
	return slices.Contains(r.Run, job) || slices.Contains(r.Run, "*")
}

// skipsJob reports whether the rule marks matched files covered for
// skipping the job.
func (r *rule) skipsJob(job string) bool {
	return slices.Contains(r.Skip, job)
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

// matches requires a root match and applies rule-local exclusions.
// Other rules can still cover the path.
func (r rule) matches(path string) bool {
	match := func(pattern *regexp.Regexp) bool {
		location := pattern.FindStringIndex(path)
		return location != nil && location[0] == 0
	}
	return slices.ContainsFunc(r.patterns, match) && !slices.ContainsFunc(r.excludedPatterns, match)
}

func fallback(err error) decision {
	return decision{log: fmt.Sprintf("selection unavailable: %v; all targets run\n", err)}
}

func (d decision) action(job string) string {
	if action := d.actions[job]; action != "" {
		return action
	}
	return "run"
}

func (d decision) String() string {
	if d.actions == nil {
		return "all targets run\n"
	}
	var text strings.Builder
	for _, job := range slices.Sorted(maps.Keys(d.actions)) {
		fmt.Fprintf(&text, "%s %s\n", job, d.actions[job])
	}
	return text.String()
}
