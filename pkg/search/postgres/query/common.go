package pgsearch

import (
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/walker"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

var (
	log = logging.LoggerForModule()
)

// SelectQueryField represents a field that's queried in a select.
type SelectQueryField struct {
	SelectPath string // This goes into the "SELECT" portion of the SQL.
	FieldType  walker.DataType
	FieldPath  string
}

// QueryEntry is an entry with clauses added by portions of the query.
type QueryEntry struct {
	Where          WhereClause
	SelectedFields []SelectQueryField
}

// WhereClause is made up of the raw query template and also the values that should be substituted
type WhereClause struct {
	Query  string
	Values []interface{}
}

// NewFalseQuery always returns false
func NewFalseQuery() *QueryEntry {
	return &QueryEntry{Where: WhereClause{
		Query: "false",
	}}
}

// NewTrueQuery always returns true
func NewTrueQuery() *QueryEntry {
	return &QueryEntry{Where: WhereClause{
		Query: "true",
	}}
}

// MatchFieldQueryFromField is a simple query that performs operations on a single field.
func MatchFieldQueryFromField(dbField *walker.Field, value string, highlight bool, optionsMap searchPkg.OptionsMap) (*QueryEntry, error) {
	if dbField == nil {
		return nil, nil
	}
	// Need to find base value
	field, ok := optionsMap.Get(dbField.Search.FieldName)
	if !ok {
		log.Infof("Options Map for %s does not have field: %v", dbField.Schema.Table, dbField.Search.FieldName)
		return nil, nil
	}
	return matchFieldQuery(dbField, field, value, highlight)
}
