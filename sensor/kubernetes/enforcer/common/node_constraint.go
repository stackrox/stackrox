package common

import (
	"fmt"
	"reflect"
)

// Label key used for unsatisfiable node constraint enforcement.
const (
	UnsatisfiableNodeConstraintKey = `BlockedByStackRoxNext`
)

// ApplyNodeConstraintToObj modifies some input type (Assuming it has a spec field) and updates it to have an
// unsatisfiable node constraint, preventing it from being scheduled.
func ApplyNodeConstraintToObj(obj interface{}, alertID string) (err error) {
	objValue := reflect.Indirect(reflect.ValueOf(obj))
	if !objValue.IsValid() || objValue.Kind() != reflect.Struct {
		return fmt.Errorf("input must have Spec field")
	}

	specValue := objValue.FieldByName("Spec")
	if !specValue.IsValid() || specValue.Kind() != reflect.Struct {
		return fmt.Errorf("input.Spec must have Template field")
	}

	templateValue := reflect.Indirect(specValue.FieldByName("Template"))
	if !templateValue.IsValid() || specValue.Kind() != reflect.Struct {
		return fmt.Errorf("input.Spec.Template must have Spec field")
	}

	podSpecValue := templateValue.FieldByName("Spec")
	if !podSpecValue.IsValid() || podSpecValue.Kind() != reflect.Struct {
		return fmt.Errorf("input.Spec.Template.Spec must have NodeSelector field")
	}

	nodeSelector := podSpecValue.FieldByName("NodeSelector")
	if nodeSelector.Kind() != reflect.Map {
		return fmt.Errorf("input.Spec.Template.Spec.NodeSelector must be map type")
	}
	if nodeSelector.IsNil() {
		nodeSelector.Set(reflect.MakeMap(nodeSelector.Type()))
	}

	nodeSelector.SetMapIndex(reflect.ValueOf(UnsatisfiableNodeConstraintKey), reflect.ValueOf(alertID))
	return
}
