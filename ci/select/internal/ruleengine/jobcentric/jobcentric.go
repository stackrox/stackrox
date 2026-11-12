// Package jobcentric is the job-centric engine behind the ruleengine interface:
// it validates the rules document, then resolves run/skip from the changed
// paths. The rule model lives in internal/rules.
package jobcentric

import (
	"github.com/stackrox/rox/ci/select/internal/resolver"
	"github.com/stackrox/rox/ci/select/internal/ruleengine"
	"github.com/stackrox/rox/ci/select/internal/validation"
)

// New returns the job-centric engine.
func New() ruleengine.Engine { return engine{} }

type engine struct{}

// Plan validates the rules document, then resolves run/skip for the changed paths.
func (engine) Plan(rules []byte, changed []string) (ruleengine.Decision, error) {
	m, err := validation.Validate(rules)
	if err != nil {
		return ruleengine.Decision{}, err
	}
	return resolver.Resolve(changed, m), nil
}
