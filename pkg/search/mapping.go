package search

import (
	"github.com/stackrox/rox/generated/api/v1"
)

// OptionMode boils down search field options into a single field
type OptionMode uint32

const (
	// OptionStore determines whether or not to store the field values in the indexer
	OptionStore OptionMode = 1 << (32 - 1 - iota)
	// OptionHidden means whether to display to the UI that the option is available
	OptionHidden
)

// NewStringField creates a new mapped field for string values.
func NewStringField(name string) *v1.SearchField {
	return NewField(name, v1.SearchDataType_SEARCH_STRING, 0)
}

// NewBoolField creates a new mapped field for boolean values.
func NewBoolField(name string) *v1.SearchField {
	return NewField(name, v1.SearchDataType_SEARCH_BOOL, 0)
}

// NewNumericField creates a new mapped field for numeric values.
func NewNumericField(name string) *v1.SearchField {
	return NewField(name, v1.SearchDataType_SEARCH_NUMERIC, 0)
}

// NewSeverityField creates a new mapped field for severity values.
func NewSeverityField(name string) *v1.SearchField {
	return NewField(name, v1.SearchDataType_SEARCH_SEVERITY, 0)
}

// NewEnforcementField creates a new mapped field for enforcement values.
func NewEnforcementField(name string) *v1.SearchField {
	return NewField(name, v1.SearchDataType_SEARCH_ENFORCEMENT, 0)
}

// NewField creates a new mapped field for any data type.
func NewField(path string, t v1.SearchDataType, mode OptionMode) *v1.SearchField {
	return &v1.SearchField{
		Type:      t,
		FieldPath: path,
		Store:     mode&OptionStore != 0,
		Hidden:    mode&OptionHidden != 0,
	}
}
