package k8s

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	v1 "k8s.io/api/core/v1"
)

var (
	log = logging.LoggerForModule()
)

// ToRoxTolerationOperator converts a v1.TolerationOperator to a storage.Toleration_Operator
func ToRoxTolerationOperator(op v1.TolerationOperator) storage.Toleration_Operator {
	switch op {
	case v1.TolerationOpExists:
		return storage.Toleration_TOLERATION_OPERATOR_EXISTS
	case v1.TolerationOpEqual:
		return storage.Toleration_TOLERATION_OPERATOR_EQUAL
	default:
		return storage.Toleration_TOLERATION_OPERATION_UNKNOWN
	}
}

// ToRoxTaintEffect converts a v1.TaintEffect to a storage.TaintEffect
func ToRoxTaintEffect(effect v1.TaintEffect) storage.TaintEffect {
	switch effect {
	case v1.TaintEffectNoSchedule:
		return storage.TaintEffect_NO_SCHEDULE_TAINT_EFFECT
	case v1.TaintEffectPreferNoSchedule:
		return storage.TaintEffect_NO_SCHEDULE_TAINT_EFFECT
	case v1.TaintEffectNoExecute:
		return storage.TaintEffect_NO_EXECUTE_TAINT_EFFECT
	default:
		return storage.TaintEffect_UNKNOWN_TAINT_EFFECT
	}
}
