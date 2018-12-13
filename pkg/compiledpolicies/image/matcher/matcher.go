package matcher

import (
	"github.com/stackrox/rox/generated/storage"
)

// Matcher is a function that provides alert violations.
type Matcher func(*storage.Image) []*storage.Alert_Violation

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

func (f orMatcherImpl) do(image *storage.Image) []*storage.Alert_Violation {
	violations1 := f.p1(image)
	violations2 := f.p2(image)

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

func (f andMatcherImpl) do(image *storage.Image) []*storage.Alert_Violation {
	violations1 := f.p1(image)
	if violations1 == nil {
		return nil
	}
	violations2 := f.p2(image)
	if violations2 == nil {
		return nil
	}
	return append(violations1, violations2...)
}
