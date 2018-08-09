package matcher

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/compiledpolicies/deployment/predicate"
)

// Matcher is a function that provides alert violations.
type Matcher func(*v1.Deployment) []*v1.Alert_Violation

// ProcessIf adds a predicate to the matcher, only executing it if the predicate passes (returns true).
func (c Matcher) ProcessIf(pred predicate.Predicate) Matcher {
	if pred == nil {
		return c
	}
	return predicatedMatcher(pred, c)
}

// CanAlsoViolate adds the input matchers output to this matchers output.
func (c Matcher) CanAlsoViolate(gen Matcher) Matcher {
	if c == nil {
		return gen
	} else if gen == nil {
		return c
	}
	return orMatcher(c, gen)
}

// MustAlsoViolate requires that the input matcher returns violations, otherwise this matcher returns no violations.
func (c Matcher) MustAlsoViolate(gen Matcher) Matcher {
	if c == nil {
		return gen
	} else if gen == nil {
		return c
	}
	return andMatcher(c, gen)
}

// orMatcherImpl provides CanAlsoViolate functionality for Matchers.
// If any Matcher returns violations, they get returned.
////////////////////////////////////////////////////////
type orMatcherImpl struct {
	p1 Matcher
	p2 Matcher
}

func orMatcher(p1, p2 Matcher) Matcher {
	return orMatcherImpl{p1, p2}.do
}

func (f orMatcherImpl) do(deployment *v1.Deployment) []*v1.Alert_Violation {
	violations1 := f.p1(deployment)
	violations2 := f.p2(deployment)

	if violations1 == nil {
		return violations2
	} else if violations2 == nil {
		return violations1
	}
	return append(violations1, violations2...)
}

// andMatcherImpl provides MustAlsoViolate functionality for Matchers.
// All Matchers must return violations or no violations are returned.
/////////////////////////////////////////////////////////////////////
type andMatcherImpl struct {
	p1 Matcher
	p2 Matcher
}

func andMatcher(p1, p2 Matcher) Matcher {
	return andMatcherImpl{p1, p2}.do
}

func (f andMatcherImpl) do(deployment *v1.Deployment) []*v1.Alert_Violation {
	violations1 := f.p1(deployment)
	if violations1 == nil {
		return nil
	}
	violations2 := f.p2(deployment)
	if violations2 == nil {
		return nil
	}
	return append(violations1, violations2...)
}

// predicatedMatcherImpl returns nil if the given predicate matches.
////////////////////////////////////////////////////////////////////
type predicatedMatcherImpl struct {
	p predicate.Predicate
	m Matcher
}

func predicatedMatcher(p predicate.Predicate, m Matcher) Matcher {
	return predicatedMatcherImpl{p, m}.do
}

func (f predicatedMatcherImpl) do(deployment *v1.Deployment) []*v1.Alert_Violation {
	if f.p(deployment) {
		return f.m(deployment)
	}
	return nil
}
