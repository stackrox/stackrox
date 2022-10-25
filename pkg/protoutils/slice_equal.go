package protoutils

import (
	"github.com/stackrox/rox/pkg/transitional/protocompat/proto"
)

// SlicesEqual returns whether the given two slices of proto objects have equal values.
func SlicesEqual[T proto.Message](first, second []T) bool {
	if len(first) != len(second) {
		return false
	}
	for i, firstElem := range first {
		secondElem := second[i]
		if !proto.Equal(firstElem, secondElem) {
			return false
		}
	}
	return true
}
