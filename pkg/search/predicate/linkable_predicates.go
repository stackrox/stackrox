package predicate

import (
	"fmt"
	"reflect"

	"github.com/stackrox/rox/pkg/search"
)

func createLinkedStructPredicate(preds ...internalPredicate) internalPredicate {
	filtered := preds[:0]
	for _, pred := range preds {
		if pred == alwaysTrue {
			continue
		}
		if pred == alwaysFalse {
			return alwaysFalse
		}
		filtered = append(filtered, pred)
	}

	if len(filtered) == 0 {
		return alwaysTrue
	}

	return internalPredicateFunc(func(instance reflect.Value) (*search.Result, bool) {
		var results []*search.Result
		for _, pred := range filtered {
			res, match := pred.Evaluate(instance)
			if !match {
				return nil, false
			}
			results = append(results, res)
		}
		return MergeResults(results...), true
	})
}

// createLinkedNestedPredicate returns a predicate that combines all of the input predicates as linked values on the
// input type.
func createLinkedNestedPredicate(fieldType reflect.Type, preds ...internalPredicate) (internalPredicate, error) {
	switch fieldType.Kind() {
	case reflect.Array, reflect.Slice:
		return createSliceLinkedPredicate(preds...), nil
	case reflect.Map:
		return createMapLinkedPredicate(preds...), nil
	case reflect.Ptr, reflect.Struct:
		return createLinkedStructPredicate(preds...), nil
	default:
		return nil, fmt.Errorf("cannot link fields within a: %s", fieldType.String())
	}
}

// Returns true if any values in the slice matches all input predicates.
func createSliceLinkedPredicate(preds ...internalPredicate) internalPredicate {
	return internalPredicateFunc(func(instance reflect.Value) (*search.Result, bool) {
		if instance.IsNil() || instance.IsZero() {
			return nil, false
		}
		var results []*search.Result
		for i := 0; i < instance.Len(); i++ {
			var localResults []*search.Result
			matchesAll := true
			for _, pred := range preds {
				res, match := pred.Evaluate(instance.Index(i))
				if !match {
					matchesAll = false
					break
				}
				localResults = append(localResults, res)
			}
			if !matchesAll {
				continue
			}
			results = append(results, localResults...)
		}
		if len(results) > 0 {
			return MergeResults(results...), true
		}
		return nil, false
	})
}

// Right now this assumes that if you are doing linked values on a map, then the key and value are the same type,
// And you don't care which one matches, as long as one of the two matches every input predicate.
func createMapLinkedPredicate(preds ...internalPredicate) internalPredicate {
	return internalPredicateFunc(func(instance reflect.Value) (*search.Result, bool) {
		if instance.IsNil() || instance.IsZero() {
			return nil, false
		}
		iter := instance.MapRange()
		for iter.Next() {
			key := iter.Key()
			val := iter.Value()

			var results []*search.Result
			matchesAll := true
			for _, pred := range preds {
				keyRes, match := pred.Evaluate(key)
				if !match {
					matchesAll = false
					break
				}
				valRes, match := pred.Evaluate(val)
				if !match {
					matchesAll = false
					break
				}
				results = append(results, keyRes, valRes)
			}
			if !matchesAll {
				continue
			}
			return MergeResults(results...), true
		}
		return nil, false
	})
}
