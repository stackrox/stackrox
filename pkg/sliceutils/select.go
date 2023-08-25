package sliceutils

import (
	"github.com/pkg/errors"
)

// Select returns a slice containing the elements at the given indices of the input slice.
// CAUTION: This function panics if any index is out of range.
func Select[T any](a []T, indices ...int) []T {
	if len(indices) == 0 {
		return nil
	}
	result := make([]T, 0, len(indices))
	for _, idx := range indices {
		if idx < 0 || idx >= len(a) {
			panic(errors.Errorf("invalid index %d: outside of expected range [0, %d)", idx, len(a)))
		}
		result = append(result, a[idx])
	}
	return result
}
