// Package resolver decides run or skip for each CI target.
// Inputs must have passed validation.Load. On one path, a skip claim
// defeats a run request.
package resolver

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/stackrox/rox/ci/select/internal/rules"
	"github.com/stackrox/rox/pkg/set"
)

// Decision carries one action ("run" or "skip") per CI target plus the
// human-readable trace explaining every decision.
type Decision struct {
	Actions map[string]string
	Log     string
}

// crossEvent records one path where a run request and a skip claim cross:
// both a run rule and a covering skip rule matched the path. Each claim
// in it is one run request and the covering skip rules that defeated it.
type crossEvent struct {
	pathNumber int
	path       string
	claims     []crossClaim
}

// crossClaim is one run request defeated by the covering skip rules of
// one path. The skip default decides every crossing the same way, so a
// claim only records who met whom; the explanation cites the covering
// rules so a reader can confirm the coverage claim.
type crossClaim struct {
	runRule    string
	coverRules []string
}

// decisionTrace is one job's verdict and the evidence behind it.
// covered is true only after every changed path contributed skip
// coverage. remaining names the rule that decided an undecided job.
type decisionTrace struct {
	action     string
	covered    bool
	coverNames set.StringSet
	runNames   set.StringSet
	runPaths   []int
	uncovered  []int
	crosses    []crossEvent
	remaining  string
}

// Resolve decides, per job, whether it runs or skips, and renders the
// traceable explanation. The pipeline inside: match rules against changed
// paths once (rules match files, not jobs), evaluate every job from those
// matches, let the remaining rule decide what nothing else did, render.
func Resolve(files []string, m rules.Mapping) Decision {
	matches := matchRules(files, m)
	traces := evaluateJobs(m, files, matches)
	fallbackToRemaining(m, traces)

	actions := make(map[string]string, len(m.Jobs))
	for _, job := range m.Jobs {
		if traces[job].action != "" {
			actions[job] = traces[job].action
		}
	}
	return Decision{Actions: actions, Log: renderTrace(files, m, matches, traces)}
}

// matchRules lists, for each changed path, the non-remaining rules that
// match it. Every job reuses this list.
func matchRules(files []string, m rules.Mapping) []fileMatches {
	matches := make([]fileMatches, 0, len(files))
	for _, path := range files {
		match := fileMatches{path: path}
		for _, r := range m.Rules {
			if r.When == rules.WhenAnyFileMatches && r.Matches(path) {
				match.rules = append(match.rules, r)
			}
		}
		matches = append(matches, match)
	}
	return matches
}

// evaluateJobs decides run, skip, or undecided for each job.
// A run request counts only on a path no skip rule covers. An empty
// diff leaves every job undecided, so the remaining rule runs it.
func evaluateJobs(m rules.Mapping, files []string, matches []fileMatches) map[string]*decisionTrace {
	traces := make(map[string]*decisionTrace, len(m.Jobs))
	for _, job := range m.Jobs {
		trace := &decisionTrace{
			coverNames: set.NewStringSet(),
			runNames:   set.NewStringSet(),
		}
		traces[job] = trace

		// An empty diff provides no paths to cover, so no job can skip:
		// every job falls through to the remaining rule below.
		if len(files) == 0 {
			continue
		}
		trace.covered = true
		for i, match := range matches {
			evaluatePath(trace, job, i+1, match)
		}
		if len(trace.runNames) > 0 {
			trace.action = "run"
		} else if trace.covered {
			trace.action = "skip"
		}
	}
	return traces
}

// evaluatePath folds one changed path into the job's trace.
// Covering skip rules on this path defeat its run requests.
func evaluatePath(trace *decisionTrace, job string, pathNumber int, match fileMatches) {
	var covers []*rules.Rule
	for _, r := range match.rules {
		if skipsJob(r, job) {
			covers = append(covers, r)
			trace.coverNames.Add(r.Name)
		}
	}
	pathCovered := len(covers) > 0

	var claims []crossClaim
	for _, r := range match.rules {
		if !requestsRun(r, job) {
			continue
		}
		if pathCovered {
			names := make([]string, 0, len(covers))
			for _, cover := range covers {
				names = append(names, cover.Name)
			}
			claims = append(claims, crossClaim{runRule: r.Name, coverRules: names})
			continue
		}
		trace.runNames.Add(r.Name)
		if !slices.Contains(trace.runPaths, pathNumber) {
			trace.runPaths = append(trace.runPaths, pathNumber)
		}
	}
	if len(claims) > 0 {
		trace.crosses = append(trace.crosses, crossEvent{
			pathNumber: pathNumber,
			path:       match.path,
			claims:     claims,
		})
	}
	// A path contributes coverage only when a skip rule covers it; a
	// covered path never counts a run request, so nothing more is needed.
	if !pathCovered {
		trace.uncovered = append(trace.uncovered, pathNumber)
	}
	trace.covered = trace.covered && pathCovered
}

// fallbackToRemaining lets the remaining rule run every job that neither
// a counting request nor full coverage decided. The rule's name is
// recorded on the trace so the explanation can cite it; the sentences
// are rendered later, in trace.go.
func fallbackToRemaining(m rules.Mapping, traces map[string]*decisionTrace) {
	for _, r := range m.Rules {
		if r.When != rules.WhenRemaining {
			continue
		}
		for _, job := range m.Jobs {
			trace := traces[job]
			if trace.action != "" {
				continue
			}
			trace.action = "run"
			trace.remaining = r.Name
		}
	}
}

// fileMatches caches the rules matching one changed path. Rules match
// files, not jobs, so the regex work is shared by every job.
type fileMatches struct {
	path  string
	rules []*rules.Rule
}

// requestsRun reports whether the rule asks to run the job on any matched
// file, including through the '*' wildcard.
func requestsRun(r *rules.Rule, job string) bool {
	return slices.Contains(r.Run, job) || slices.Contains(r.Run, "*")
}

// skipsJob reports whether the rule marks matched files covered for
// skipping the job.
func skipsJob(r *rules.Rule, job string) bool {
	return slices.Contains(r.Skip, job)
}

// Matches on rules.Rule (in the rules package) holds the production
// prefix-match semantics the resolver evaluates each path with.

func Fallback(err error) Decision {
	return Decision{Log: fmt.Sprintf("selection unavailable: %v; all targets run\n", err)}
}

func (d Decision) Action(job string) string {
	if action := d.Actions[job]; action != "" {
		return action
	}
	return "run"
}

func (d Decision) String() string {
	if d.Actions == nil {
		return "all targets run\n"
	}
	var text strings.Builder
	for _, job := range slices.Sorted(maps.Keys(d.Actions)) {
		fmt.Fprintf(&text, "%s %s\n", job, d.Actions[job])
	}
	return text.String()
}
