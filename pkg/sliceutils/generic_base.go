package sliceutils

import "github.com/pkg/errors"

// ElemTypeSelect returns a slice containing the elements at the given indices of the input slice.
// CAUTION: This function panics if any index is out of range.
func ElemTypeSelect(a []ElemType, indices ...int) []ElemType {
	result := make([]ElemType, 0, len(indices))
	for _, idx := range indices {
		if idx < 0 || idx >= len(a) {
			panic(errors.Errorf("invalid index %d: outside of expected range [0, %d)", idx, len(a)))
		}
		result = append(result, a[idx])
	}
	return result
}

// ElemTypeClone clones a slice, creating a new slice
// and copying the contents of the underlying array.
// If `in` is a nil slice, a nil slice is returned.
// If `in` is an empty slice, an empty slice is returned.
func ElemTypeClone(in []ElemType) []ElemType {
	if in == nil {
		return nil
	}
	if len(in) == 0 {
		return []ElemType{}
	}
	out := make([]ElemType, len(in))
	copy(out, in)
	return out
}

// ConcatElemTypeSlices concatenates slices, returning a slice with newly allocated backing storage of the exact
// size.
func ConcatElemTypeSlices(slices ...[]ElemType) []ElemType {
	length := 0
	for _, slice := range slices {
		length += len(slice)
	}
	result := make([]ElemType, length)
	i := 0
	for _, slice := range slices {
		nextI := i + len(slice)
		copy(result[i:nextI], slice)
		i = nextI
	}
	return result
}

//go:generate genny -in=$GOFILE -out=gen-builtins-$GOFILE gen "ElemType=BUILTINS,ByteSlice"
