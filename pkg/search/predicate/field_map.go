package predicate

import (
	"strings"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

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

type wrappedOptionsMap struct {
	optionsMap search.OptionsMap
	prefix     string
}

func (w wrappedOptionsMap) Get(field string) (*v1.SearchField, bool) {
	searchField, ok := w.optionsMap.Get(field)
	if !ok {
		return nil, false
	}
	if !strings.HasPrefix(searchField.GetFieldPath(), w.prefix) {
		return nil, false
	}
	return searchField, true
}
