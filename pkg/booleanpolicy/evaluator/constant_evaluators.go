package evaluator

import (
	"github.com/stackrox/stackrox/pkg/booleanpolicy/evaluator/pathutil"
)

type alwaysTrueType struct{}

func (alwaysTrueType) Evaluate(pathutil.AugmentedValue) (*Result, bool) {
	return &Result{}, true
}

var (
	// AlwaysTrue is an evaluator that always returns true.
	AlwaysTrue Evaluator = alwaysTrueType{}
)
