package regexengine

import (
	"fmt"
	"slices"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/ci/select/internal/rules"
	"github.com/stackrox/rox/pkg/set"
)

func checkStructure(m rules.Mapping) error {
	if len(m.Jobs) == 0 || len(m.Rules) == 0 {
		return errors.New("rules need nonempty jobs and rules lists")
	}
	jobs := set.NewStringSet()
	for _, job := range m.Jobs {
		valid := job != "*" && len(strings.Fields(job)) == 1 &&
			strings.TrimSpace(job) == job && !strings.Contains(job, "#")
		if !valid || !jobs.Add(job) {
			return fmt.Errorf("invalid or duplicate job %q", job)
		}
	}
	names := set.NewStringSet()
	for i, r := range m.Rules {
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
		if err := checkTargets(r, jobs); err != nil {
			return err
		}
		if err := checkPatterns(r); err != nil {
			return err
		}
	}
	return nil
}

func checkTargets(r *rules.Rule, jobs set.StringSet) error {
	seen := set.NewStringSet()
	for _, targets := range [][]string{r.Run, r.Skip} {
		for _, job := range targets {
			if job == "*" {
				if len(r.Skip) > 0 || !slices.Equal(r.Run, []string{"*"}) {
					return fmt.Errorf("rule %q supports the wildcard only as run: ['*'], without skips", r.Name)
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

func checkPatterns(r *rules.Rule) error {
	included := false
	for _, pattern := range r.Paths {
		expression, exclude := strings.CutPrefix(pattern, "!")
		if !strings.HasPrefix(expression, "^") {
			return fmt.Errorf("rule %q pattern %q must be rooted with ^", r.Name, pattern)
		}
		included = included || !exclude
	}
	if !included {
		return fmt.Errorf("rule %q must include at least one non-excluded path pattern", r.Name)
	}
	return nil
}
