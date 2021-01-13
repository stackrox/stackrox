package maputil

import (
	"reflect"

	"github.com/stackrox/rox/pkg/reflectutils"
	"github.com/stackrox/rox/pkg/set"
)

// DiffLeaf contains the information for a diff found within untyped maps.
type DiffLeaf struct {
	A interface{} // This is the value in the `a` resource.
	B interface{} // This is the value in the `b` resource.
}

// DiffGenericMap computes a diff in the form of a generic map for two generic maps.
// The values in the result map are either of type `map[string]interface{}` or `DiffLeaf`.
func DiffGenericMap(a map[string]interface{}, b map[string]interface{}) map[string]interface{} {
	keys := set.NewStringSet()
	diffMap := make(map[string]interface{})

	// collect all keys first
	for k := range a {
		keys.Add(k)
	}
	for k := range b {
		keys.Add(k)
	}

	// Compute any diffs for the given objets by traversing the union of the keys.
	for _, k := range keys.AsSortedSlice(func(a, b string) bool { return a < b }) {
		aVal := a[k]
		bVal := b[k]

		aMap, aOk := aVal.(map[string]interface{})
		bMap, bOk := bVal.(map[string]interface{})
		if aOk && bOk {
			// Compute diffs for the nested maps.
			subDiff := DiffGenericMap(aMap, bMap)
			if subDiff != nil {
				diffMap[k] = subDiff
			}
		} else if reflectutils.IsNil(aVal) && reflectutils.IsNil(bVal) {
			continue
		} else if reflect.TypeOf(aVal) != reflect.TypeOf(bVal) || !reflect.DeepEqual(aVal, bVal) {
			// Type or value mismatch.
			diffMap[k] = &DiffLeaf{
				A: aVal,
				B: bVal,
			}
		}
	}

	if len(diffMap) == 0 {
		return nil
	}

	return diffMap
}
