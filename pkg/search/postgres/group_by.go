package postgres

import (
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/postgres/walker"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
)

func populateGroupBy(querySoFar *query, groupBy *v1.QueryGroupBy, schema *walker.Schema, queryFields map[string]searchFieldMetadata) error {
	if querySoFar.QueryType != SELECT && len(groupBy.GetFields()) > 0 {
		return errors.New("GROUP BY clause not supported with SEARCH query type; Use SELECT")
	}

	// If explicit group by clauses are not specified and if a query field (in select or order by) is a derived field requiring a group by clause,
	// default to primary key grouping. Note that all fields in the query, including pagination, are in `queryFields`.
	if len(groupBy.GetFields()) == 0 {
		for _, field := range queryFields {
			if field.derivedMetadata == nil {
				continue
			}
			switch field.derivedMetadata.DerivationType {
			case searchPkg.CountDerivationType:
				applyGroupByPrimaryKeys(querySoFar, schema)
				return nil
			}
		}
		return nil
	}

	for _, groupByField := range groupBy.GetFields() {
		fieldMetadata := queryFields[groupByField]
		dbField := fieldMetadata.baseField
		if dbField == nil {
			return errors.Errorf("field %s in GROUP BY clause does not exist in table %s or connected tables", groupByField, schema.Table)
		}
		if fieldMetadata.derivedMetadata != nil {
			// Aggregate functions are not allowed in GROUP BY clause. SQL constraint.
			return errors.Errorf("found %s in GROUP BY clause. Derived fields cannot be used in GROUP BY clause", groupByField)
		}
		if dbField.Options.PrimaryKey {
			querySoFar.GroupByPrimaryKey = true
		}

		selectField := selectQueryField(groupByField, dbField, false, aggregatefunc.Unset, "")
		selectField.FromGroupBy = true
		querySoFar.GroupBys = append(querySoFar.GroupBys, groupByEntry{Field: selectField})
	}
	return nil
}

func applyGroupByPrimaryKeys(querySoFar *query, schema *walker.Schema) {
	querySoFar.GroupByPrimaryKey = true
	pks := schema.PrimaryKeys()
	for idx := range pks {
		pk := &pks[idx]
		selectField := selectQueryField("", pk, false, aggregatefunc.Unset, "")
		selectField.FromGroupBy = true
		querySoFar.GroupBys = append(querySoFar.GroupBys, groupByEntry{Field: selectField})
	}
}
