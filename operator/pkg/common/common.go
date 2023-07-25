package common

import "github.com/stackrox/rox/pkg/env"

var (
	//OperatorOuterMode represents whether the Operator manages Central & the inner operator (outer mode)
	// 					or Secured Clusters (inner mode)
	OperatorOuterMode = env.RegisterBooleanSetting("ROX_OPERATOR_OUTER_MODE", true)
)
