package predicate

import (
	"github.com/stackrox/rox/generated/api/v1"
)

// Compiler is a function that takes in a policy and returns a prefdicte that returns TRUE if the policy should
// be evaluated on the input.
type compiler func(*v1.Policy) (Predicate, error)

// compilers are all of the different Predicate creation functions that are registered.
var compilers []compiler

// Compile creates a new container predicate for the input policy.
func Compile(policy *v1.Policy) (Predicate, error) {
	var pred Predicate
	for _, compiler := range compilers {
		shouldProcessFunction, err := compiler(policy)
		if err != nil {
			return nil, err
		}
		pred = pred.And(shouldProcessFunction)
	}
	return pred, nil
}
