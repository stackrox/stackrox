package search

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
)

// Field describes a search field
type Field struct {
	FieldPath string
	Type      v1.SearchDataType
	Store     bool
	Hidden    bool
	Category  v1.SearchCategory
	Analyzer  string
}

// GetFieldPath returns the field path
func (f *Field) GetFieldPath() string {
	if f == nil {
		return ""
	}
	return f.FieldPath
}

// GetType returns the type
func (f *Field) GetType() v1.SearchDataType {
	if f == nil {
		return 0
	}
	return f.Type
}

// GetStore returns whether or not the data is stored
func (f *Field) GetStore() bool {
	if f == nil {
		return false
	}
	return f.Store
}

// GetHidden returns whether or not the option is shown to users
func (f *Field) GetHidden() bool {
	if f == nil {
		return false
	}
	return f.Hidden
}

// GetCategory returns the search category
func (f *Field) GetCategory() v1.SearchCategory {
	if f == nil {
		return 0
	}
	return f.Category
}

// GetAnalyzer returns the search analyzer for the field
func (f *Field) GetAnalyzer() string {
	if f == nil {
		return ""
	}
	return f.Analyzer
}
