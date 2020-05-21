package predicate

import (
	"strings"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

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
