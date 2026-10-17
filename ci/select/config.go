package main

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
)

type rule struct {
	Name     string   `toml:"name"`
	When     string   `toml:"when"`
	Paths    []string `toml:"paths"`
	Label    string   `toml:"label"`
	Run      []string `toml:"run"`
	Skip     []string `toml:"skip"`
	Default  []string `toml:"default"`
	patterns []*regexp.Regexp
}

type mapping struct {
	Version  int                 `toml:"version"`
	Jobs     []string            `toml:"jobs"`
	Rules    []rule              `toml:"rules"`
	Requires map[string][]string `toml:"requires"`
}

func loadMapping(path string) (mapping, error) {
	var result mapping
	meta, err := toml.DecodeFile(path, &result)
	if err != nil {
		return result, errors.Wrap(err, "read TOML rules")
	}
	if meta.IsDefined("requires") && meta.Type("requires") != "Hash" {
		return result, errors.New("requires must be a table")
	}
	return result, result.validate()
}

func (m *mapping) validate() error {
	if m.Version != 1 {
		return fmt.Errorf("unsupported rules version: %d", m.Version)
	}
	if len(m.Jobs) == 0 {
		return errors.New("jobs must list at least one target")
	}
	jobs := map[string]bool{}
	for _, job := range m.Jobs {
		if jobs[job] {
			return fmt.Errorf("duplicate job name %s", job)
		}
		if job == "" || strings.ContainsAny(job, "\r\n#") {
			return fmt.Errorf("bad job name %q", job)
		}
		jobs[job] = true
	}
	if len(m.Rules) == 0 {
		return errors.New("rules must be a non-empty list")
	}
	names := map[string]bool{}
	for index := range m.Rules {
		r := &m.Rules[index]
		if err := r.validate(index+1, jobs); err != nil {
			return err
		}
		if names[r.Name] {
			return fmt.Errorf("duplicate rule name %s", r.Name)
		}
		names[r.Name] = true
		if r.When == "remaining" && index != len(m.Rules)-1 {
			return fmt.Errorf("rule %d %s uses remaining, which only the last rule may use", index+1, r.Name)
		}
	}
	last := m.Rules[len(m.Rules)-1]
	if last.When != "remaining" {
		return errors.New("the last rule must decide every target no earlier rule decided")
	}
	if !slices.Contains(last.Run, "*") && !slices.Contains(last.Skip, "*") && !slices.Contains(last.Default, "*") {
		return errors.New("the last rule must run, skip, or take the default for every remaining target")
	}
	for job, required := range m.Requires {
		if !jobs[job] {
			return fmt.Errorf("unknown job %s in requires", job)
		}
		if len(required) == 0 {
			return fmt.Errorf("%s requires a non-empty list", job)
		}
		seen := map[string]bool{}
		for _, dependency := range required {
			if !jobs[dependency] {
				return fmt.Errorf("unknown job %s required by %s", dependency, job)
			}
			if dependency == job {
				return fmt.Errorf("%s requires itself", job)
			}
			if seen[dependency] {
				return fmt.Errorf("duplicate requirement for %s", job)
			}
			seen[dependency] = true
		}
	}
	return m.checkCycles()
}

func (r *rule) validate(index int, jobs map[string]bool) error {
	where := fmt.Sprintf("rule %d %s", index, r.Name)
	if r.Name == "" {
		return fmt.Errorf("rule %d needs a name", index)
	}
	pathRule := r.When == "any-file-matches" || r.When == "every-file-matches"
	switch r.When {
	case "any-file-matches", "every-file-matches", "always", "label-exists", "no-file-changed", "remaining":
	default:
		return fmt.Errorf("%s has unknown when: %q", where, r.When)
	}
	if pathRule && len(r.Paths) == 0 {
		return fmt.Errorf("%s needs paths", where)
	}
	if !pathRule && len(r.Paths) != 0 {
		return fmt.Errorf("%s does not take paths", where)
	}
	if r.When == "label-exists" && r.Label == "" {
		return fmt.Errorf("%s needs a label", where)
	}
	if r.When != "label-exists" && r.Label != "" {
		return fmt.Errorf("%s does not take a label", where)
	}
	seen := map[string]bool{}
	starred := 0
	for _, list := range [][]string{r.Run, r.Skip, r.Default} {
		local := map[string]bool{}
		for _, job := range list {
			if job == "*" {
				if len(list) != 1 {
					return fmt.Errorf("%s mixes * with job names", where)
				}
				starred++
				continue
			}
			if !jobs[job] {
				return fmt.Errorf("unknown job %s in %s", job, where)
			}
			if local[job] {
				return fmt.Errorf("%s repeats a job", where)
			}
			if seen[job] {
				return fmt.Errorf("%s gives one job two opinions", where)
			}
			local[job], seen[job] = true, true
		}
	}
	if starred > 1 {
		return fmt.Errorf("%s uses * more than once", where)
	}
	if starred != 0 && len(seen) != 0 {
		return fmt.Errorf("%s mixes * with job names", where)
	}
	if starred == 0 && len(seen) == 0 {
		return fmt.Errorf("%s must run, skip, or take the default", where)
	}
	r.patterns = make([]*regexp.Regexp, 0, len(r.Paths))
	for _, pattern := range r.Paths {
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("invalid pattern %q in %s: %w", pattern, where, err)
		}
		r.patterns = append(r.patterns, compiled)
	}
	return nil
}

func (m mapping) checkCycles() error {
	visiting := map[string]bool{}
	visited := map[string]bool{}
	var walk func(string) error
	walk = func(job string) error {
		if visiting[job] {
			return fmt.Errorf("requires cycle at %s", job)
		}
		if visited[job] {
			return nil
		}
		visiting[job] = true
		for _, dependency := range m.Requires[job] {
			if err := walk(dependency); err != nil {
				return err
			}
		}
		delete(visiting, job)
		visited[job] = true
		return nil
	}
	for _, job := range m.Jobs {
		if err := walk(job); err != nil {
			return err
		}
	}
	return nil
}
