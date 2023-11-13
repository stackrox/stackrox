package fieldmap

import (
	"reflect"
	"strings"
)

// FieldPath represents the fields we need to access to get to the field we care about.
type FieldPath []reflect.StructField

// FieldMap is a wrapper for a map from search option to field path to compare.
type FieldMap map[string]FieldPath

// Add adds a key/value pair to the map.
func (fm FieldMap) Add(k string, fp FieldPath) {
	fm[strings.ToLower(k)] = fp
}

// Get returns a key from the map if present.
func (fm FieldMap) Get(k string) FieldPath {
	return fm[strings.ToLower(k)]
}
