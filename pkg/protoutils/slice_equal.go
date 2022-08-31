package protoutils

import (
	"github.com/mauricelam/genny/generic"
)

// ProtoSliceType represents a generic type that we use in the function below.
//
//go:generate genny -in=$GOFILE -imp=github.com/stackrox/rox/generated/storage -out=gen-$GOFILE gen "ProtoSliceType=*storage.Alert_Violation"
type ProtoSliceType generic.Type

// EqualProtoSliceTypeSlices returns whether the given two slices of proto objects (generically) have equal values.
func EqualProtoSliceTypeSlices(first, second []ProtoSliceType) bool {
	if len(first) != len(second) {
		return false
	}
	for i, firstElem := range first {
		secondElem := second[i]
		if !protoEqualWrapper(firstElem, secondElem) {
			return false
		}
	}
	return true
}
