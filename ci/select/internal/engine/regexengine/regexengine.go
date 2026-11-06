// Package regexengine implements engine.Engine for today's rules
// language: a YAML mapping of anchored regex path patterns that
// request runs or mark skip coverage, decided by the resolver.
package regexengine

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/ci/select/internal/engine"
	"github.com/stackrox/rox/ci/select/internal/resolver"
	"github.com/stackrox/rox/ci/select/internal/rules"
)

// New returns the engine for the current rules language.
func New() engine.Engine {
	return engineImpl{}
}

type engineImpl struct{}

// compiled is this engine's private engine.Rules representation.
type compiled struct{ m rules.Mapping }

// Compile parses the document, enforces the version this engine
// implements, and compiles the patterns. The version check is the
// engine's own, not a validator's: an engine must never guess the
// meaning of a document it does not know, so trivial validation still
// fails closed here instead of mis-selecting.
func (engineImpl) Compile(data []byte) (engine.Rules, error) {
	m, err := rules.Parse(data)
	if err != nil {
		return nil, errors.Wrap(err, "parse rules")
	}
	if m.Version != 1 {
		return nil, fmt.Errorf("unsupported rules version %d, only 1 is supported", m.Version)
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
