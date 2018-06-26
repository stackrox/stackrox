package search

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

// NewStringField creates a new mapped field for string values.
func NewStringField(name string) *v1.SearchField {
	return NewField(name, v1.SearchDataType_SEARCH_STRING, false)
}

// NewBoolField creates a new mapped field for boolean values.
func NewBoolField(name string) *v1.SearchField {
	return NewField(name, v1.SearchDataType_SEARCH_BOOL, false)
}

// NewNumericField creates a new mapped field for numeric values.
func NewNumericField(name string) *v1.SearchField {
	return NewField(name, v1.SearchDataType_SEARCH_NUMERIC, false)
}

// NewSeverityField creates a new mapped field for severity values.
func NewSeverityField(name string) *v1.SearchField {
	return NewField(name, v1.SearchDataType_SEARCH_SEVERITY, false)
}

// NewEnforcementField creates a new mapped field for enforcement values.
func NewEnforcementField(name string) *v1.SearchField {
	return NewField(name, v1.SearchDataType_SEARCH_ENFORCEMENT, false)
}

// NewField creates a new mapped field for any data type.
func NewField(path string, t v1.SearchDataType, store bool) *v1.SearchField {
	return &v1.SearchField{
		Type:      t,
		FieldPath: path,
		Store:     store,
	}
}
