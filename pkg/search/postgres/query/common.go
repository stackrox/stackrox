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

	// PostTransform is a function that will be applied to the returned rows from SQL before
	// further processing.
	// The input will be of the type directly returned from the postgres rows.Scan function.
	// The output must be of the same type as the input.
	// It will be nil if there is no transform to be applied.
	PostTransform func(interface{}) interface{}
}

// QueryEntry is an entry with clauses added by portions of the query.
type QueryEntry struct {
	Where          WhereClause
	SelectedFields []SelectQueryField

	// This is populated only in the case of enums, so that callers know how to
	// convert the returned enum value to a string.
	// This is a bit ugly, but enums are such a special-case that it seems
	// better to treat them explicitly rather than making this a more generic
	// post-process func.
	enumStringifyFunc func(int32) string
}

// WhereClause is made up of the raw query template and also the values that should be substituted
type WhereClause struct {
	Query  string
	Values []interface{}

	// equivalentGoFunc returns the equivalent Go function to the Where clause.
	// It is used in cases where we may want to do post-filtering.
	// It will not always be set.
	equivalentGoFunc func(foundValue interface{}) bool
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
