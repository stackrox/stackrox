// Package rules holds the job-selection rule model and path matching.
//
// The model is job-centric: every job is governed by exactly one rule (or by
// always_run). That single-owner invariant is the whole point — it removes
// cross-rule conflicts, so there is nothing to resolve or detect at runtime.
package rules

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"

	"github.com/pkg/errors"
	"go.yaml.in/yaml/v3"
)

type (
	// Mapping is one parsed rules file. Jobs is the inventory; AlwaysRun and
	// Rules together must decide each inventory job exactly once.
	Mapping struct {
		Version   int      `json:"version"`
		Jobs      []string `json:"jobs"`
		AlwaysRun []string `json:"always_run"`
		Rules     []*Rule  `json:"rules"`
	}

	Rule struct {
		Name              string   `json:"name"`
		Jobs              []string `json:"jobs"`
		RunIfChanged      []string `json:"run_if_changed"`
		SkipIfOnlyChanged []string `json:"skip_if_only_changed"`

		runPatterns  []*matcher
		skipPatterns []*matcher
	}

	// matcher carries the '!'-exclusion flag alongside the compiled regexp.
	matcher struct {
		expression *regexp.Regexp
		exclude    bool
	}
)

// Parse decodes one YAML document into a Mapping. It rejects a malformed
// document and leaves rule substance to validation.
func Parse(data []byte) (Mapping, error) {
	var m Mapping
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
	// Round-trip through JSON for strict typing: YAML would coerce a number or
	// bool into a string field, and an unknown key must fail rather than be
	// silently ignored and change selection.
	strictData, err := json.Marshal(document)
	if err != nil {
		return m, errors.Wrap(err, "convert rules to JSON")
	}
	strict := json.NewDecoder(bytes.NewReader(strictData))
	strict.DisallowUnknownFields()
	if err := strict.Decode(&m); err != nil {
		return m, errors.Wrap(err, "decode rules fields")
	}
	return m, nil
}

// Compile must run before any matching; an uncompiled rule matches nothing.
func (r *Rule) Compile() error {
	run, err := compilePatterns(r.Name, r.RunIfChanged)
	if err != nil {
		return err
	}
	skip, err := compilePatterns(r.Name, r.SkipIfOnlyChanged)
	if err != nil {
		return err
	}
	r.runPatterns = run
	r.skipPatterns = skip
	return nil
}

// RunMatchesAny reports whether any changed file matches run_if_changed;
// false when the rule declares no run_if_changed patterns.
func (r *Rule) RunMatchesAny(files []string) bool {
	if len(r.runPatterns) == 0 {
		return false
	}
	return slices.ContainsFunc(files, func(path string) bool {
		return matches(r.runPatterns, path)
	})
}

// SkipCoversAll reports whether every changed file matches skip_if_only_changed;
// false when the rule declares no such patterns or the diff is empty (an empty
// diff is never "only" anything, so it cannot justify a skip).
func (r *Rule) SkipCoversAll(files []string) bool {
	if len(r.skipPatterns) == 0 || len(files) == 0 {
		return false
	}
	return !slices.ContainsFunc(files, func(path string) bool {
		return !matches(r.skipPatterns, path)
	})
}

func (r *Rule) HasRun() bool  { return len(r.RunIfChanged) > 0 }
func (r *Rule) HasSkip() bool { return len(r.SkipIfOnlyChanged) > 0 }

func compilePatterns(ruleName string, sources []string) ([]*matcher, error) {
	compiled := make([]*matcher, 0, len(sources))
	for _, source := range sources {
		expression, exclude := strings.CutPrefix(source, "!")
		pattern, err := regexp.Compile(expression)
		if err != nil {
			return nil, fmt.Errorf("rule %q pattern %q: %w", ruleName, source, err)
		}
		compiled = append(compiled, &matcher{expression: pattern, exclude: exclude})
	}
	return compiled, nil
}

// matches is prefix-anchored (location[0] == 0): a pattern must match from the
// start of the path but need not consume it, so '^central/' matches any file
// under central/. An exclusion match vetoes an inclusion match.
func matches(patterns []*matcher, path string) bool {
	included, excluded := false, false
	for _, pattern := range patterns {
		if location := pattern.expression.FindStringIndex(path); location != nil && location[0] == 0 {
			if pattern.exclude {
				excluded = true
			} else {
				included = true
			}
		}
	}
	return included && !excluded
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
