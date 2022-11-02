package compiler

import (
	"fmt"

	"github.com/itchyny/gojq"
	"github.com/pkg/errors"
	"github.com/stackrox/helmtest/internal/logic"
	"gopkg.in/yaml.v3"
)

// Additional built-ins for gojq

func fromYaml(obj interface{}, _ []interface{}) interface{} {
	var bytes []byte
	switch o := obj.(type) {
	case []byte:
		bytes = o
	case string:
		bytes = []byte(o)
	default:
		return errors.Errorf("expected string or bytes as input to fromyaml, got %T", obj)
	}

	var out map[string]interface{}
	if err := yaml.Unmarshal(bytes, &out); err != nil {
		return errors.Wrap(err, "fromyaml")
	}
	return out
}

func toYaml(obj interface{}, _ []interface{}) interface{} {
	bytes, err := yaml.Marshal(obj)
	if err != nil {
		return errors.Wrap(err, "toyaml")
	}
	return string(bytes)
}

func printjq(obj interface{}, _ []interface{}) interface{} {
	fmt.Println("--------DEBUG--------")
	fmt.Println(obj)
	return obj
}

func assertThat(obj interface{}, args []interface{}) interface{} {
	evalResult := args[0]
	if !logic.Truthy(evalResult) {
		return errors.Errorf("%+v failed predicate '%s'", obj, args[1])
	}
	return true
}

func assumeThat(obj interface{}, args []interface{}) interface{} {
	evalResult := args[0]
	if !logic.Truthy(evalResult) {
		return logic.ErrAssumptionViolation
	}
	return obj
}

func assertNotExist(obj interface{}, _ []interface{}) interface{} {
	return errors.Errorf("object %+v should not exist", obj)
}

var builtinOpts = []gojq.CompilerOption{
	gojq.WithFunction("fromyaml", 0, 0, fromYaml),
	gojq.WithFunction("toyaml", 0, 0, toYaml),
	gojq.WithFunction("assertThat", 2, 2, assertThat),
	gojq.WithFunction("assertNotExist", 0, 0, assertNotExist),
	gojq.WithFunction("assumeThat", 1, 1, assumeThat),
	gojq.WithFunction("print", 0, 0, printjq),
}
