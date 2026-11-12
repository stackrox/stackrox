// Package regexengine implements engine.Engine for today's rules
// language: a YAML mapping of anchored regex path patterns that
// request runs or mark skip coverage, decided by the resolver.
package regexengine

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/ci/select/internal/engine"
	"github.com/stackrox/rox/ci/select/internal/engine/regexengine/rulesvalidation"
	"github.com/stackrox/rox/ci/select/internal/resolver"
	"github.com/stackrox/rox/ci/select/internal/rules"
)

// New returns the engine for the current rules language with
// syntax-only checks: the document must be one well-formed YAML
// document, nothing more.
func New() engine.Engine {
	return newEngine(rulesvalidation.YAML{})
}

// NewTrivial returns the same language with authoring checks disabled.
// Compilation still rejects documents the engine cannot evaluate, so the
// only thing lowered is the rule-quality report.
func NewTrivial() engine.Engine {
	return newEngine(engine.TrivialValidator{})
}

func newEngine(checks engine.Validator) engine.Engine {
	return engineImpl{checks: checks}
}

type engineImpl struct {
	checks engine.Validator
}

// compiled is this engine's private engine.Rules representation.
type compiled struct{ m rules.Mapping }

// Validate runs this language's checks on one document. The error is
// the rendered report the CLI prints verbatim, so it passes through
// unwrapped.
func (e engineImpl) Validate(data []byte) ([]string, error) {
	return e.checks.Validate(data) //nolint:wrapcheck
}

// Compile enforces the language's structure and patterns independently of
// optional authoring checks, so invalid rules cannot silently alter selection.
func (engineImpl) Compile(data []byte) (engine.Rules, error) {
	m, err := rules.Parse(data)
	if err != nil {
		return nil, errors.Wrap(err, "parse rules")
	}
	if m.Version != 1 {
		return nil, fmt.Errorf("unsupported rules version %d, only 1 is supported", m.Version)
	}
	if err := checkStructure(m); err != nil {
		return nil, errors.Wrap(err, "check rules structure")
	}
	for _, r := range m.Rules {
		if err := r.Compile(); err != nil {
			return nil, errors.Wrap(err, "compile patterns")
		}
	}
	return compiled{m}, nil
}

// Resolve delegates the decision to the resolver, which owns this
// language's run/skip semantics.
func (engineImpl) Resolve(files []string, rules engine.Rules) engine.Decision {
	c, ok := rules.(compiled)
	if !ok {
		return engine.Fallback(errors.New("rules compiled by another engine"))
	}
	return resolver.Resolve(files, c.m)
}
