package sliceutils

import "fmt"

// StringSlice returns a string slice from the given T.
func StringSlice[T fmt.Stringer](in ...T) []string {
	res := make([]string, 0, len(in))
	for _, i := range in {
		res = append(res, i.String())
	}
	return res
}

// FromStringSlice returns a slice T from the given strings.
// Note that this only works for types whose underlying type is string.
func FromStringSlice[T ~string](in ...string) []T {
	res := make([]T, 0, len(in))
	for _, i := range in {
		res = append(res, T(i))
	}
	return res
}
