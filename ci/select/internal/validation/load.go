// Package validation lints a job-selection rules file. The job-centric model
// makes cross-rule contradictions impossible, so only structural checks
// remain (see checkStructure) — no pattern-overlap analysis.
package validation

import (
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/ci/select/internal/rules"
)

// Load reads, parses, and checks the rules file. On success the returned
// Mapping has compiled patterns ready for the resolver.
func Load(path string) (rules.Mapping, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return rules.Mapping{}, errors.Wrap(err, "read rules")
	}
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
