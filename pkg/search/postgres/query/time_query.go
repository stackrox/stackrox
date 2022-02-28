package pgsearch

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/search"
)

func newTimeQuery(table string, field *search.Field, value string, modifiers ...search.QueryModifier) (*QueryEntry, error) {
	return nil, errors.New("time queries are currently unimplemented")
}
