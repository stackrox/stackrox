package evaluator

import (
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator/pathutil"
)

type alwaysTrueType struct{}

func (alwaysTrueType) Evaluate(pathutil.AugmentedValue) (*Result, bool) {
	return &Result{}, true
}

var (
	alwaysTrue Evaluator = alwaysTrueType{}
)
