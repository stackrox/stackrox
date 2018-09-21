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

// NewStoredTimeField creates a new mapped field for timestamp values we want to store.
func NewStoredTimeField(category v1.SearchCategory, name string) *v1.SearchField {
	return NewField(category, name, v1.SearchDataType_SEARCH_DATETIME, OptionStore)
}

// NewTimeField creates a new mapped field for timestamp values
func NewTimeField(category v1.SearchCategory, name string) *v1.SearchField {
	return NewField(category, name, v1.SearchDataType_SEARCH_DATETIME, 0)
}

// NewStoredStringField creates a new mapped field for string values we want to store.
func NewStoredStringField(category v1.SearchCategory, name string) *v1.SearchField {
	return NewField(category, name, v1.SearchDataType_SEARCH_STRING, OptionStore)
}

// NewStringField creates a new mapped field for string values.
func NewStringField(category v1.SearchCategory, name string) *v1.SearchField {
	return NewField(category, name, v1.SearchDataType_SEARCH_STRING, 0)
}

// NewMapField creates a new mapped field for a map
func NewMapField(category v1.SearchCategory, name string) *v1.SearchField {
	return NewField(category, name, v1.SearchDataType_SEARCH_MAP, 0)
}

// NewStoredBoolField creates a new mapped field for boolean values we want to store.
func NewStoredBoolField(category v1.SearchCategory, name string) *v1.SearchField {
	return NewField(category, name, v1.SearchDataType_SEARCH_BOOL, OptionStore)
}

// NewBoolField creates a new mapped field for boolean values.
func NewBoolField(category v1.SearchCategory, name string) *v1.SearchField {
	return NewField(category, name, v1.SearchDataType_SEARCH_BOOL, 0)
}

// NewStoredNumericField creates a new mapped field for numeric values we want to store.
func NewStoredNumericField(category v1.SearchCategory, name string) *v1.SearchField {
	return NewField(category, name, v1.SearchDataType_SEARCH_NUMERIC, OptionStore)
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

// NewSecretTypeField creates a new mapped field for secret type enum
func NewSecretTypeField(category v1.SearchCategory, name string) *v1.SearchField {
	return NewField(category, name, v1.SearchDataType_SEARCH_SECRET_TYPE, 0)
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
