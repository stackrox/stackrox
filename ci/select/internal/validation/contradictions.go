package validation

// contradictions.go reports claims that need no diff to be wrong.
// A crossing only some diffs hit stays a warning in conflicts.go.
// The errors are a dead inclusion, the same include text used to run
// and to skip, and a run pattern one skip rule covers entirely.

import (
	"fmt"
	"slices"
	"strings"

	"github.com/stackrox/rox/ci/select/internal/overlap"
	"github.com/stackrox/rox/ci/select/internal/rules"
	"github.com/stackrox/rox/pkg/set"
)

// checkContradictions merges the detectors into one sorted report,
// dropping duplicate lines repeated patterns within a rule produce.
func checkContradictions(m rules.Mapping) []string {
	seen := set.NewStringSet()
	var errs []string
	lines := append(deadInclusions(m.Rules), duplicatedPatterns(m.Rules)...)
	for _, line := range append(lines, coveredPatterns(m.Rules)...) {
		if !seen.Add(line) {
			continue
		}
		errs = append(errs, line)
	}
	slices.Sort(errs)
	return errs
}

// deadInclusions finds inclusion patterns a rule's own exclusions fully cover.
func deadInclusions(all []*rules.Rule) []string {
	var errs []string
	for _, r := range all {
		if r.When != rules.WhenAnyFileMatches {
			continue
		}
		includes, excludes := r.PathSources()
		if len(excludes) == 0 {
			continue
		}
		for _, pattern := range includes {
			dead, example := inclusionIsDead(pattern, excludes)
			if !dead {
				continue
			}
			detail := ""
			if example != "" {
				detail = fmt.Sprintf(" (e.g. %q)", example)
			}
			errs = append(errs, fmt.Sprintf(
				"rule %q: pattern %q matches only files its own exclusions cover%s",
				r.Name, pattern, detail))
		}
	}
	slices.Sort(errs)
	return errs
}

// inclusionIsDead reports whether pattern matches only files excludes cover.
// A conservative automaton reports a survivor, so a dead pattern can be
// missed and is never invented.
func inclusionIsDead(pattern string, excludes []string) (bool, string) {
	escaped := make([]string, 0, len(excludes)+1)
	escaped = append(escaped, "(?s:.*)")
	for _, exclusion := range excludes {
		escaped = append(escaped, "!"+exclusion)
	}
	escaper := &rules.Rule{Paths: escaped}
	if err := escaper.Compile(); err != nil {
		return false, ""
	}
	target := &rules.Rule{Paths: []string{pattern}}
	if err := target.Compile(); err != nil {
		return false, ""
	}
	// The pattern is dead when it misses every file outside the exclusions.
	if overlap.Of(target, escaper).Meets {
		return false, ""
	}
	witness := overlap.Of(target, &rules.Rule{Paths: excludes}).Witness
	if witness != "" && target.Matches(witness) && exclusionMatches(excludes, witness) {
		return true, witness
	}
	return true, ""
}

// exclusionMatches reports whether any raw exclusion expression
// prefix-matches the path — the r rule's own check without its
// inclusions, which the dead pattern cannot satisfy.
func exclusionMatches(excludes []string, path string) bool {
	for _, exclusion := range excludes {
		probe := &rules.Rule{Paths: []string{exclusion}}
		if err := probe.Compile(); err != nil {
			continue
		}
		if probe.Matches(path) {
			return true
		}
	}
	return false
}

// duplicatedPatterns finds the same include text used to run a job in
// one rule and skip it in another, when the two copies still match a
// common file. The probe is that string plus the rule's own exclusions,
// so a different pattern on the same rule cannot cause the hit. An
// inexact automaton is not proof, and the pair stays a warning.
func duplicatedPatterns(all []*rules.Rule) []string {
	type claim struct {
		rule    *rules.Rule
		pattern string
	}
	var runs, skips []claim
	for _, r := range all {
		if r.When != rules.WhenAnyFileMatches {
			continue
		}
		includes, _ := r.PathSources()
		for _, pattern := range includes {
			if len(r.Run) > 0 {
				runs = append(runs, claim{r, pattern})
			}
			if len(r.Skip) > 0 {
				skips = append(skips, claim{r, pattern})
			}
		}
	}
	var errs []string
	for _, run := range runs {
		for _, skip := range skips {
			if run.pattern != skip.pattern || run.rule == skip.rule {
				continue
			}
			jobs := overlappingTargets(run.rule.Run, skip.rule.Skip)
			if len(jobs) == 0 {
				continue
			}
			_, runExcludes := run.rule.PathSources()
			_, skipExcludes := skip.rule.PathSources()
			runProbe := patternProbe(run.pattern, runExcludes)
			skipProbe := patternProbe(skip.pattern, skipExcludes)
			meeting := overlap.Of(runProbe, skipProbe)
			if !meeting.Meets || !meeting.Exact {
				continue
			}
			detail := ""
			if example := trustedWitness(runProbe, skipProbe, meeting.Witness); example != "" {
				detail = fmt.Sprintf(" and both match %q", example)
			}
			errs = append(errs, fmt.Sprintf(
				"rules %q and %q both use %q for %s%s",
				run.rule.Name, skip.rule.Name, run.pattern, joinQuoted(jobs), detail))
		}
	}
	return errs
}

// coveredPatterns finds a run pattern whose files one skip rule all
// matches. No diff can make that pattern count. The same include text is
// left to duplicatedPatterns. An inexact automaton is not proof.
func coveredPatterns(all []*rules.Rule) []string {
	var runs, skips []*rules.Rule
	for _, r := range all {
		if r.When != rules.WhenAnyFileMatches {
			continue
		}
		if len(r.Run) > 0 {
			runs = append(runs, r)
		}
		if len(r.Skip) > 0 {
			skips = append(skips, r)
		}
	}
	var errs []string
	for _, run := range runs {
		includes, excludes := run.PathSources()
		for _, pattern := range includes {
			probe := patternProbe(pattern, excludes)
			for _, skip := range skips {
				if run == skip {
					continue
				}
				jobs := overlappingTargets(run.Run, skip.Skip)
				if len(jobs) == 0 {
					continue
				}
				if slices.Contains(skipIncludes(skip), pattern) {
					continue
				}
				cover := overlap.Covers(skip, probe)
				if !cover.Exact || !cover.Covers {
					continue
				}
				// An empty pattern is covered by everything and is reported
				// as a dead inclusion instead. Coverage here needs a file.
				meeting := overlap.Of(probe, skip)
				if !meeting.Meets || !meeting.Exact {
					continue
				}
				detail := ""
				if example := trustedWitness(probe, skip, meeting.Witness); example != "" {
					detail = fmt.Sprintf(" (e.g. %q)", example)
				}
				errs = append(errs, fmt.Sprintf(
					"rule %q: pattern %q never runs %s; %q matches every file of that pattern%s",
					run.Name, pattern, joinQuoted(jobs), skip.Name, detail))
			}
		}
	}
	return errs
}

func skipIncludes(r *rules.Rule) []string {
	includes, _ := r.PathSources()
	return includes
}

// patternProbe is one include plus its rule's exclusions, the effective
// file set of that line alone.
func patternProbe(pattern string, excludes []string) *rules.Rule {
	paths := make([]string, 0, len(excludes)+1)
	paths = append(paths, pattern)
	for _, exclusion := range excludes {
		paths = append(paths, "!"+exclusion)
	}
	probe := &rules.Rule{Paths: paths}
	_ = probe.Compile()
	return probe
}

// trustedWitness keeps witness only when both rules match it. The
// automaton is a second matcher, so a report cites a path only after
// the production matcher agrees. Lowercase is preferred when it also matches.
func trustedWitness(a, b *rules.Rule, witness string) string {
	if witness == "" || !a.Matches(witness) || !b.Matches(witness) {
		return ""
	}
	if lower := strings.ToLower(witness); lower != witness && a.Matches(lower) && b.Matches(lower) {
		return lower
	}
	return witness
}
