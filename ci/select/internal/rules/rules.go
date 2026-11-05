// Package rules holds the job-selection rule model and how one rule
// matches one path. Whether a rules file is valid is validation's job.
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

// Rule kinds. A rule either applies when any changed file matches its
// path patterns, or it is the final fallback for everything neither run
// nor covered.
const (
	WhenAnyFileMatches = "any-file-matches"
	WhenRemaining      = "remaining"
)

type (
	Mapping struct {
		Version int      `json:"version"`
		Jobs    []string `json:"jobs"`
		Rules   []*Rule  `json:"rules"`
	}

	Rule struct {
		Name             string   `json:"name"`
		When             string   `json:"when"`
		Paths            []string `json:"paths"`
		Run              []string `json:"run"`
		Skip             []string `json:"skip"`
		patterns         []*regexp.Regexp
		excludedPatterns []*regexp.Regexp
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
	// JSON enforces scalar types; YAML decoding into strings also accepts
	// numbers and booleans. Unknown fields must not silently alter selection.
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

// PathSources splits path patterns on the '!' exclusion prefix.
// The automata and the production matchers both start from this split.
func (r Rule) PathSources() (includes, excludes []string) {
	for _, pattern := range r.Paths {
		if expression, excluded := strings.CutPrefix(pattern, "!"); excluded {
			excludes = append(excludes, expression)
		} else {
			includes = append(includes, pattern)
		}
	}
	return includes, excludes
}

// includesPrefix reports a match that starts at index 0. The pattern
// does not have to consume the rest of the path.
func (r Rule) includesPrefix(path string) bool {
	return slices.ContainsFunc(r.patterns, func(pattern *regexp.Regexp) bool {
		location := pattern.FindStringIndex(path)
		return location != nil && location[0] == 0
	})
}

// Matches reports whether path is included and not excluded by this rule.
// Validation trusts a witness only when both rules of a pair Matches it.
func (r Rule) Matches(path string) bool {
	excluded := slices.ContainsFunc(r.excludedPatterns, func(pattern *regexp.Regexp) bool {
		location := pattern.FindStringIndex(path)
		return location != nil && location[0] == 0
	})
	return r.includesPrefix(path) && !excluded
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

// Compile builds the matchers Matches uses, splitting '!' exclusions
// the same way as PathSources. An uncompiled rule matches nothing.
func (r *Rule) Compile() error {
	for _, pattern := range r.Paths {
		expression, exclude := strings.CutPrefix(pattern, "!")
		compiled, err := regexp.Compile(expression)
		if err != nil {
			return fmt.Errorf("rule %q pattern %q: %w", r.Name, pattern, err)
		}
		if exclude {
			r.excludedPatterns = append(r.excludedPatterns, compiled)
		} else {
			r.patterns = append(r.patterns, compiled)
		}
	}
	return nil
}
