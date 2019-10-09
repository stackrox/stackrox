package predicate

import (
	"fmt"
	"reflect"
)

// createLinkedNestedPredicate returns a predicate that combines all of the input predicates as linked values on the
// input type.
func createLinkedNestedPredicate(fieldType reflect.Type, preds ...internalPredicate) (internalPredicate, error) {
	switch fieldType.Kind() {
	case reflect.Array:
		return createSliceLinkedPredicate(preds...), nil
	case reflect.Slice:
		return createSliceLinkedPredicate(preds...), nil
	case reflect.Map:
		return createMapLinkedPredicate(preds...), nil
	default:
		return nil, fmt.Errorf("cannot link fields within a: %s", fieldType.String())
	}
}

// Returns true if any values in the slice matches all input predicates.
func createSliceLinkedPredicate(preds ...internalPredicate) internalPredicate {
	return func(instance reflect.Value) bool {
		if instance.IsNil() || instance.IsZero() {
			return false
		}
		for i := 0; i < instance.Len(); i++ {
			passedAllPreds := true
			for _, pred := range preds {
				if !pred(instance.Index(i)) {
					passedAllPreds = false
					break
				}
			}
			if passedAllPreds {
				return true
			}
		}
		return false
	}
}

// Right now this assumes that if you are doing linked values on a map, then the key and value are the same type,
// And you don't care which one matches, as long as one of the two matches every input predicate.
func createMapLinkedPredicate(preds ...internalPredicate) internalPredicate {
	return func(instance reflect.Value) bool {
		if instance.IsNil() || instance.IsZero() {
			return false
		}
		iter := instance.MapRange()
		for iter.Next() {
			key := iter.Key()
			val := iter.Value()

			passedAllPreds := true
			for _, pred := range preds {
				if !pred(key) && !pred(val) {
					passedAllPreds = false
					break
				}
			}
			if passedAllPreds {
				return true
			}
		}
		return false
	}
}
