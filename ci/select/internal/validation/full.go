// Package validation lints a job-selection rules file. Validator is
// the pluggable entry; Full, the default implementation, returns
// warnings for run/skip overlaps and an error when the file contradicts
// itself.
package validation

import (
	"fmt"
	"slices"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/ci/select/internal/rules"
	"github.com/stackrox/rox/pkg/set"
)

// Full checks every finding today's authoring workflow knows: file
// structure, target and pattern defects, contradictions that need no
// diff to be wrong, and run/skip conflicts, in one numbered report.
type Full struct{}

// Validate runs the whole check pipeline on one document. Contradictions
// — a pattern excluded by its own rule, the same include text used to
// run and skip a job when those copies still meet, or a run pattern one
// skip rule covers entirely — are errors. A run/skip overlap that some
// diffs miss is a warning.
func (Full) Validate(data []byte) ([]string, error) {
	m, err := rules.Parse(data)
	if err != nil {
		return nil, errors.Wrap(err, "parse rules")
	}
	if err := checkStructure(m); err != nil {
		return nil, err
	}
	for _, r := range m.Rules {
		if err := r.Compile(); err != nil {
			return nil, errors.Wrap(err, "compile patterns")
		}
	}
	return report(m)
}

// report numbers every finding, errors before warnings, in one continuous
// sequence so each has a citable reference, and appends the footer to the
// warnings. It returns the joined error lines only.
func report(m rules.Mapping) ([]string, error) {
	var errs, warnings []string
	n := 0
	for _, line := range checkContradictions(m) {
		n++
		errs = append(errs, fmt.Sprintf("%03d Error: %s", n, line))
	}
	for _, line := range Check(m) {
		n++
		warnings = append(warnings, fmt.Sprintf("%03d Warning: %s", n, line))
	}
	errCount, warningCount := len(errs), len(warnings)
	if errCount+warningCount > 0 {
		// A blank line separates the findings from the totals; an empty
		// report gets no separator.
		warnings = append(warnings, "")
	}
	warnings = append(warnings, findingsFooter(errCount, warningCount))
	if errCount > 0 {
		return warnings, errors.New(strings.Join(errs, "\n"))
	}
	return warnings, nil
}

// findingsFooter states the report totals. Skip is named only when a
// warning needs a winner, and resolver failure only when an error does.
func findingsFooter(errCount, warningCount int) string {
	if errCount == 0 && warningCount == 0 {
		return "Result: no issues found."
	}
	text := fmt.Sprintf("Result: %d %s, %d %s.",
		errCount, plural(errCount, "error"),
		warningCount, plural(warningCount, "warning"))
	if warningCount > 0 {
		text += fmt.Sprintf(" The %q wins all warning conflicts.", "skip")
	}
	if errCount > 0 {
		text += " Errors fail the resolver."
	}
	return text
}

func plural(n int, singular string) string {
	if n == 1 {
		return singular
	}
	return singular + "s"
}

// checkStructure verifies the file-level invariants: known and unrepeated
// targets, unique rule names, the final remaining rule, and the
// anchored-pattern conventions.
func checkStructure(m rules.Mapping) error {
	if m.Version != 1 {
		return fmt.Errorf("unsupported rules version %d, only 1 is supported", m.Version)
	}
	if len(m.Jobs) == 0 {
		return errors.New("rules need a nonempty jobs list")
	}
	if len(m.Rules) == 0 {
		return errors.New("rules need a nonempty rules list")
	}
	jobs := set.NewStringSet()
	for _, job := range m.Jobs {
		if !isJobNameValid(job) {
			return fmt.Errorf("invalid job name %q", job)
		}
		if !jobs.Add(job) {
			return fmt.Errorf("duplicate job %q", job)
		}
	}
	names := set.NewStringSet()
	for i, r := range m.Rules {
		if r == nil {
			return fmt.Errorf("rule %d is null", i+1)
		}
		if r.Name == "" || !names.Add(r.Name) {
			return fmt.Errorf("rule %d needs a unique nonempty name", i+1)
		}
		if i == len(m.Rules)-1 {
			if r.When != rules.WhenRemaining || !slices.Equal(r.Run, []string{"*"}) || len(r.Skip)+len(r.Paths) != 0 {
				return errors.New("last rule must run ['*'] for remaining targets, without paths or skips")
			}
			continue
		}
		if r.When != rules.WhenAnyFileMatches || len(r.Paths) == 0 || len(r.Skip)+len(r.Run) == 0 {
			return fmt.Errorf("rule %q must use any-file-matches with paths and run or skip targets", r.Name)
		}
		if err := checkRuleTargets(r, jobs); err != nil {
			return err
		}
		if err := checkRulePatterns(r); err != nil {
			return err
		}
	}
	return nil
}

func checkRuleTargets(r *rules.Rule, jobs set.StringSet) error {
	seen := set.NewStringSet()
	for _, targets := range [][]string{r.Skip, r.Run} {
		for _, job := range targets {
			if job == "*" {
				if len(r.Skip) > 0 || !slices.Equal(r.Run, []string{"*"}) {
					return fmt.Errorf("rule %q: the wildcard is supported only as run: ['*'], without skips or other targets", r.Name)
				}
				continue
			}
			if !jobs.Contains(job) {
				return fmt.Errorf("rule %q names unknown target %q", r.Name, job)
			}
			if !seen.Add(job) {
				return fmt.Errorf("rule %q repeats target %q within one list or across run and skip", r.Name, job)
			}
		}
	}
	return nil
}

// checkRulePatterns enforces the anchoring convention (patterns must root
// with ^) and rejects rules with nothing left to match. Compilation
// itself happens in Compile right after; this runs first so an unrooted
// pattern is reported as a rule defect, not a regex error.
func checkRulePatterns(r *rules.Rule) error {
	included := 0
	for _, pattern := range r.Paths {
		expression, exclude := strings.CutPrefix(pattern, "!")
		if !strings.HasPrefix(expression, "^") {
			return fmt.Errorf("rule %q pattern %q must be rooted with ^", r.Name, pattern)
		}
		if !exclude {
			included++
		}
	}
	if included == 0 {
		return fmt.Errorf("rule %q must include at least one non-excluded path pattern", r.Name)
	}
	return nil
}

func isJobNameValid(job string) bool {
	return job != "*" &&
		len(strings.Fields(job)) == 1 &&
		strings.TrimSpace(job) == job &&
		!strings.Contains(job, "#")
}
