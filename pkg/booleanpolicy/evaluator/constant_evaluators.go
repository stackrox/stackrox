package evaluator

import (
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator/pathutil"
)

type alwaysTrueType struct{}

func (alwaysTrueType) Evaluate(_ *pathutil.AugmentedObj) (*Result, bool) {
	return &Result{}, true
}

var (
	// AlwaysTrue is an evaluator that always returns true.
	AlwaysTrue Evaluator = alwaysTrueType{}
)
