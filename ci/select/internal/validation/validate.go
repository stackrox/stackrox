// Package validation checks a rules file for structural problems (see
// checkStructure). The single-owner model needs no cross-rule overlap analysis.
package validation

import (
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/ci/select/internal/rules"
)

// ValidateFile reads the rules file, then validates it with Validate.
func ValidateFile(path string) (rules.Mapping, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return rules.Mapping{}, errors.Wrap(err, "read rules")
	}
	return Validate(data)
}

// Validate checks rules already in memory — the byte entry point for the engine
// interface. On success the Mapping has compiled patterns ready for the resolver.
func Validate(data []byte) (rules.Mapping, error) {
	m, err := rules.Parse(data)
	if err != nil {
		return m, errors.Wrap(err, "parse rules")
	}
	if findings := checkStructure(m); len(findings) > 0 {
		return m, numbered(findings)
	}
	for _, r := range m.Rules {
		if err := r.Compile(); err != nil {
			return m, errors.Wrap(err, "compile patterns")
		}
	}
	return m, nil
}

// numbered reports all findings at once (not just the first) so an author
// fixing the rules file sees everything in one run.
func numbered(findings []string) error {
	lines := make([]string, len(findings))
	for i, finding := range findings {
		lines[i] = fmt.Sprintf("%03d Error: %s", i+1, finding)
	}
	return errors.New(strings.Join(lines, "\n"))
}
