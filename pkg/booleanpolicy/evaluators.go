package booleanpolicy

import (
	"strings"

	"github.com/stackrox/rox/pkg/config"
	"github.com/stackrox/rox/pkg/set"
)

// EvaluatorType is the type of evaluator.
type EvaluatorType int

const (
	// Legacy is the current engine
	Legacy EvaluatorType = iota
	// RegoBase is the OPA rego with outter join
	RegoBase
	// RegoOr is the OPA rego with inner join
	RegoOr
	// RegoNegate is the OPA rego with inner join and negate operation
	RegoNegate
	// Cel is the Common Expression Language
	Cel
)

func getConfiguredEvaluatorTypes() set.Set[EvaluatorType] {
	conf := config.GetConfig()
	if conf.Maintenance.PolicyEvaluators == "" {
		// By default, test legacy, Rego and CEL
		// return set.NewSet[EvaluatorType](Legacy, RegoOr, Cel)
		return set.NewSet[EvaluatorType](Legacy)
	}

	// Split the string into individual evaluator type names
	evaluatorTypeNames := strings.Split(conf.Maintenance.PolicyEvaluators, ",")

	// Create a map to map evaluator type names to their respective EvaluatorType constants
	evaluatorTypeMap := map[string]EvaluatorType{
		"legacy":     Legacy,
		"regoBase":   RegoBase,
		"regoOr":     RegoOr,
		"regoNegate": RegoNegate,
		"cel":        Cel,
	}

	// Create a set (map) to store the selected EvaluatorType values
	evalSet := set.NewSet[EvaluatorType](Legacy)

	// Iterate through the evaluator type names and add them to the set
	for _, name := range evaluatorTypeNames {
		// Check if the name exists in the map
		if val, ok := evaluatorTypeMap[name]; ok {
			_ = evalSet.Add(val)
		} else {
			log.Errorf("Unknown evaluator type: %s\n", name)
		}
	}
	return evalSet
}
