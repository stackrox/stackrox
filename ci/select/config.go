package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/set"
	"go.yaml.in/yaml/v3"
)

type (
	mapping struct {
		Version int      `json:"version"`
		Jobs    []string `json:"jobs"`
		Rules   []rule   `json:"rules"`
	}

	rule struct {
		Name             string   `json:"name"`
		When             string   `json:"when"`
		Paths            []string `json:"paths"`
		Run              []string `json:"run"`
		Skip             []string `json:"skip"`
		patterns         []*regexp.Regexp
		excludedPatterns []*regexp.Regexp
	}
)

// loadMapping reads and validates the job-selection rules file at the given path.
// JSON re-decoding with DisallowUnknownFields enforces scalar types and rejects
// unknown fields; validation of the semantics lives in validate.
func loadMapping(path string) (mapping, error) {
	var m mapping
	data, err := os.ReadFile(path)
	if err != nil {
		return m, errors.Wrap(err, "read rules")
	}
	var document map[string]any
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&document); err != nil {
		return m, errors.Wrap(err, "decode rules")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return m, errors.New("rules must contain a single YAML document")
	}
	if err := rejectNull(document); err != nil {
		return m, err
	}
	// JSON enforces scalar types; YAML decoding into strings also accepts
	// numbers and booleans. Unknown fields must not silently alter selection.
	data, err = json.Marshal(document)
	if err != nil {
		return m, errors.Wrap(err, "convert rules to JSON")
	}
	strict := json.NewDecoder(bytes.NewReader(data))
	strict.DisallowUnknownFields()
	if err := strict.Decode(&m); err != nil {
		return m, errors.Wrap(err, "decode rules fields")
	}
	return m, m.validate()
}

func rejectNull(value any) error {
	switch value := value.(type) {
	case nil:
		return errors.New("rules must not contain null values")
	case map[string]any:
		for _, child := range value {
			if err := rejectNull(child); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range value {
			if err := rejectNull(child); err != nil {
				return err
			}
		}
	}
	return nil
}

func isJobNameValid(job string) bool {
	return job != "*" &&
		len(strings.Fields(job)) == 1 &&
		strings.TrimSpace(job) == job &&
		!strings.Contains(job, "#")
}

func (m *mapping) validate() error {
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
	for i := range m.Rules {
		r := &m.Rules[i]
		if r.Name == "" || !names.Add(r.Name) {
			return fmt.Errorf("rule %d needs a unique nonempty name", i+1)
		}
		if i == len(m.Rules)-1 {
			if r.When != "remaining" || !slices.Equal(r.Run, []string{"*"}) || len(r.Skip)+len(r.Paths) != 0 {
				return errors.New("last rule must run ['*'] for remaining targets, without paths or skips")
			}
			continue
		}
		if r.When != "any-file-matches" || len(r.Paths) == 0 || len(r.Skip)+len(r.Run) == 0 {
			return fmt.Errorf("rule %q must use any-file-matches with paths and run or skip targets", r.Name)
		}
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
		for _, pattern := range r.Paths {
			expression, exclude := strings.CutPrefix(pattern, "!")
			if !strings.HasPrefix(expression, "^") {
				return fmt.Errorf("rule %q pattern %q must be rooted with ^", r.Name, pattern)
			}
			compiled, err := regexp.Compile(expression)
			if err != nil {
				return fmt.Errorf("rule %q pattern %q: %w", r.Name, pattern, err)
			}
			if exclude {
				r.excludedPatterns = append(r.excludedPatterns, compiled)
				continue
			}
			r.patterns = append(r.patterns, compiled)
		}
		if len(r.patterns) == 0 {
			return fmt.Errorf("rule %q must include at least one non-excluded path pattern", r.Name)
		}
	}
	return nil
}
