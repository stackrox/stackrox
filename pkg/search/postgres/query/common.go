package pgsearch

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/search"
)

var (
	log = logging.LoggerForModule()
)

// SelectQueryField represents a field that's queried in a select.
type SelectQueryField struct {
	SelectPath string // This goes into the "SELECT" portion of the SQL.
	Alias      string // Alias for "SelectPath". Primarily used for derived fields.
	FieldType  postgres.DataType
	FieldPath  string // This is the search.Field.FieldPath for this field.

	// FromGroupBy indicates that the field is present in group by clause.
	FromGroupBy bool
	// DerivedField indicates that the field is derived from a proto field(/table column).
	DerivedField bool

	// PostTransform is a function that will be applied to the returned rows from SQL before
	// further processing.
	// The input will be of the type directly returned from the postgres rows.Scan function.
	// It will be nil if there is no transform to be applied.
	// Currently, the PostTransform is only used on arrays, so that we only highlight values
	// that match the queries.
	// For example, if we have the following table:
	// key | string_array_column
	// 0   | {"ab", "abc", "cd"}
	// 1   | {"xyz"}
	// And we have a query that's looking for a prefix of "a".
	// If we do "select string_array_column from table where string_array_column like 'a%'", we get
	// key | string_array_column
	// 0   | {"ab", "abc", "cd"}
	// However, this includes "cd", which does NOT match the query, which is not ideal for highlights.
	// Therefore, we do a post-transform where we apply the query to the returned rows in Go,
	// so that the final result seen by the user includes only {"ab", "abc"}.
	PostTransform func(interface{}) interface{}
}

// PathForSelectPortion returns the selector string that goes into SELECT portion of the SQL.
func (f *SelectQueryField) PathForSelectPortion() string {
	if f.Alias == "" {
		return f.SelectPath
	}
	return fmt.Sprintf("%s as %s", f.SelectPath, f.Alias)
}

// QueryEntry is an entry with clauses added by portions of the query.
type QueryEntry struct {
	Where          WhereClause
	Having         *WhereClause
	SelectedFields []SelectQueryField
	GroupBy        []string

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
	// It is used in cases where we want to do some post-processing in Go space
	// because doing it in SQL is too hairy.
	// See the documentation of PostTransform in SelectQueryField for more details.
	// It will not always be set.
	equivalentGoFunc func(foundValue interface{}) bool
}

// NewFalseQuery always returns false
func NewFalseQuery() *QueryEntry {
	return &QueryEntry{Where: WhereClause{
		Query:            "false",
		equivalentGoFunc: func(_ interface{}) bool { return false },
	}}
}

// NewTrueQuery always returns true
func NewTrueQuery() *QueryEntry {
	return &QueryEntry{Where: WhereClause{
		Query: "true",
	}}
}

// MatchFieldQuery is a simple query that performs operations on a single field.
func MatchFieldQuery(dbField *walker.Field, derivedMetadata *walker.DerivedSearchField, value string, highlight bool, now time.Time) (*QueryEntry, error) {
	if dbField == nil {
		return nil, nil
	}
	// Need to find base value
	if dbField.Schema.OptionsMap == nil {
		return nil, errors.Errorf("Options Map for %s does not exist", dbField.Schema.Table)
	}
	field, ok := dbField.Schema.OptionsMap.Get(dbField.Search.FieldName)
	if !ok {
		return nil, nil
	}

	qualifiedColName := dbField.Schema.Table + "." + dbField.ColumnName
	dataType := dbField.DataType
	if dbField.SQLType == "uuid" {
		dataType = postgres.UUID
	}
	if derivedMetadata != nil {
		switch derivedMetadata.DerivationType {
		case search.CountDerivationType:
			qualifiedColName = fmt.Sprintf("count(%s)", qualifiedColName)
			dataType = postgres.Integer
		default:
			return nil, errors.Errorf("unsupported derivation type %s", derivedMetadata.DerivationType)
		}
	}
	qe, err := matchFieldQuery(qualifiedColName, dataType, field, derivedMetadata, value, highlight, now)
	if err != nil {
		return nil, err
	}
	return qe, nil
}
