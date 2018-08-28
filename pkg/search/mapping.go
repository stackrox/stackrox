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

// NewTimeField creates a new mapped field for timestamp values
func NewTimeField(category v1.SearchCategory, name string) *v1.SearchField {
	return NewField(category, name, v1.SearchDataType_SEARCH_DATETIME, 0)
}

// NewStringField creates a new mapped field for string values.
func NewStringField(category v1.SearchCategory, name string) *v1.SearchField {
	return NewField(category, name, v1.SearchDataType_SEARCH_STRING, 0)
}

// NewMapField creates a new mapped field for a map
func NewMapField(category v1.SearchCategory, name string) *v1.SearchField {
	return NewField(category, name, v1.SearchDataType_SEARCH_MAP, 0)
}

// NewBoolField creates a new mapped field for boolean values.
func NewBoolField(category v1.SearchCategory, name string) *v1.SearchField {
	return NewField(category, name, v1.SearchDataType_SEARCH_BOOL, 0)
}

// NewNumericField creates a new mapped field for numeric values.
func NewNumericField(category v1.SearchCategory, name string) *v1.SearchField {
	return NewField(category, name, v1.SearchDataType_SEARCH_NUMERIC, 0)
}

// NewSeverityField creates a new mapped field for severity values.
func NewSeverityField(category v1.SearchCategory, name string) *v1.SearchField {
	return NewField(category, name, v1.SearchDataType_SEARCH_SEVERITY, 0)
}

// NewEnforcementField creates a new mapped field for enforcement values.
func NewEnforcementField(category v1.SearchCategory, name string) *v1.SearchField {
	return NewField(category, name, v1.SearchDataType_SEARCH_ENFORCEMENT, 0)
}

// NewField creates a new mapped field for any data type.
func NewField(category v1.SearchCategory, path string, t v1.SearchDataType, mode OptionMode) *v1.SearchField {
	return &v1.SearchField{
		Category:  category,
		Type:      t,
		FieldPath: path,
		Store:     mode&OptionStore != 0,
		Hidden:    mode&OptionHidden != 0,
	}
}
