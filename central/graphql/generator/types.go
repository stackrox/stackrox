package generator

import "reflect"

// TypeWalkParameters tells generator how to walk types
type TypeWalkParameters struct {
	IncludedTypes []reflect.Type
}
