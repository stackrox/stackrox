package pgsearch

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/walker"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

var (
	log = logging.LoggerForModule()
)

// QueryEntry is made up of the raw query template and also the values that should be substituted
type QueryEntry struct {
	Query  string
	Values []interface{}
}

// NewFalseQuery always returns false
func NewFalseQuery() *QueryEntry {
	return &QueryEntry{
		Query: "false",
	}
}

// NewTrueQuery always returns true
func NewTrueQuery() *QueryEntry {
	return &QueryEntry{
		Query: "true",
	}
}

// MatchFieldQuery is a simple query that performs operations on a single field.
func MatchFieldQuery(schema *walker.Schema, query *v1.MatchFieldQuery, optionsMap searchPkg.OptionsMap) (*QueryEntry, error) {
	// Need to find base value
	field, ok := optionsMap.Get(query.GetField())
	if !ok {
		log.Infof("Options Map for %s does not have field: %v", schema.Table, query.GetField())
		return nil, nil
	}
	fieldsBySearchLabel := schema.FieldsBySearchLabel()
	dbField := fieldsBySearchLabel[query.GetField()]
	if dbField == nil {
		log.Errorf("Missing field %s in table %s", query.GetField(), schema.Table)
		return nil, nil
	}
	return matchFieldQuery(dbField, field, query.Value)
}

// MatchFieldQueryFromField is a simple query that performs operations on a single field.
func MatchFieldQueryFromField(dbField *walker.Field, value string, optionsMap searchPkg.OptionsMap) (*QueryEntry, error) {
	if dbField == nil {
		return nil, nil
	}
	// Need to find base value
	// TODO: A query can span across multiple tables. Therefore, the primary options map may not contain the query field.
	//  Consider embedding field path into  walker.Field.
	field, ok := optionsMap.Get(dbField.Search.FieldName)
	if !ok {
		log.Infof("Options Map for %s does not have field: %v", dbField.Schema.Table, dbField.Search.FieldName)
		return nil, nil
	}
	return matchFieldQuery(dbField, field, value)
}
