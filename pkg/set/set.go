package set

import (
	"github.com/deckarep/golang-set"
)

// NewSetFromStringSlice returns a new set from the elements of the slice
func NewSetFromStringSlice(strs []string) mapset.Set {
	newSet := mapset.NewSet()
	for _, s := range strs {
		newSet.Add(s)
	}
	return newSet
}

// StringSliceFromSet converts a Set to a slice of strings
func StringSliceFromSet(s mapset.Set) []string {
	strs := make([]string, 0, s.Cardinality())
	for str := range s.Iter() {
		strs = append(strs, str.(string))
	}
	return strs
}

// AppendStringMapKeys adds all keys of the passed map[string]string to the set
func AppendStringMapKeys(s mapset.Set, m map[string]string) {
	for k := range m {
		s.Add(k)
	}
}
