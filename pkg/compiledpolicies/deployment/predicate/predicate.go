package predicate

import (
	"github.com/stackrox/rox/generated/storage"
)

// Predicate is a function that indicates whether (true) or not (false) a deployment should be processed.
type Predicate func(*storage.Deployment) bool

// Or returns a new Predicate that is the logical OR of this Predicate and the given Predicate.
func (c Predicate) Or(gen Predicate) Predicate {
	if c == nil {
		return gen
	} else if gen == nil {
		return c
	}
	return orPredicate(c, gen)
}

// And returns a new Predicate that is the logical AND of this Predicate and the given Predicate.
func (c Predicate) And(gen Predicate) Predicate {
	if c == nil {
		return gen
	} else if gen == nil {
		return c
	}
	return andPredicate(c, gen)
}

// orPredicateImpl provides OR functionality for Predicates.
// If any Predicate returns violations, they get returned.
///////////////////////////////////////////////////////////////////
type orPredicateImpl struct {
	p1 Predicate
	p2 Predicate
}

func orPredicate(p1, p2 Predicate) Predicate {
	return orPredicateImpl{p1, p2}.do
}

func (f orPredicateImpl) do(deployment *storage.Deployment) bool {
	return f.p1(deployment) || f.p2(deployment)
}

// andPredicateImpl provides AND functionality for Predicates.
// All Predicates must return violations or no violations are returned.
////////////////////////////////////////////////////////////////////////////////
type andPredicateImpl struct {
	p1 Predicate
	p2 Predicate
}

func andPredicate(p1, p2 Predicate) Predicate {
	return andPredicateImpl{p1, p2}.do
}

func (f andPredicateImpl) do(deployment *storage.Deployment) bool {
	return f.p1(deployment) && f.p2(deployment)
}
