package derivelocalvalues

import (
	"reflect"

	"github.com/stackrox/rox/pkg/set"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// DiffLeaf contains the information for a diff found within untyped maps.
type DiffLeaf struct {
	A interface{} // This is the value in the `a` resource.
	B interface{} // This is the value in the `b` resource.
}

// For the given untyped maps compute a diff in the form of another untyped map.
// The values in the result map are either of type `map[string]interface{}` or `DiffLeaf`.
func diff(a map[string]interface{}, b map[string]interface{}) map[string]interface{} {
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
			subDiff := diff(aMap, bMap)
			if subDiff != nil {
				diffMap[k] = subDiff
			}
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

func diffUnstructured(a unstructured.Unstructured, b unstructured.Unstructured) map[string]interface{} {
	return diff(a.UnstructuredContent(), b.UnstructuredContent())
}
