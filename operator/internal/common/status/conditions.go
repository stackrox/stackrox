package status

import (
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
)

var (
	statusControllerConditionTypes = []platform.ConditionType{
		platform.ConditionAvailable,
		platform.ConditionProgressing,
	}
)
