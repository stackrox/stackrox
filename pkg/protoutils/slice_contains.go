package protoutils

import "github.com/mauricelam/genny/generic"

// ProtoType represents a generic type that we use in the function below.
//go:generate genny -in=$GOFILE -imp=github.com/stackrox/rox/generated/storage -out=gen-$GOFILE gen "ProtoType=*storage.Signature"
type ProtoType generic.Type

// ContainsProtoTypeInSlice returns whether the given proto object is contained in the given slice.
func ContainsProtoTypeInSlice(proto ProtoType, slice []ProtoType) bool {
	for _, elem := range slice {
		if protoEqualWrapper(proto, elem) {
			return true
		}
	}
	return false
}
