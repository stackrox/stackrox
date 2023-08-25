package postgres

import (
	"fmt"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
	pgsearch "github.com/stackrox/rox/pkg/search/postgres/query"
)

func populatePagination(querySoFar *query, pagination *v1.QueryPagination, schema *walker.Schema, queryFields map[string]searchFieldMetadata) error {
	if pagination == nil {
		return nil
	}

	for idx, so := range pagination.GetSortOptions() {
		if idx != 0 && so.GetSearchAfter() != "" {
			return errors.New("search after for pagination must be defined for only the first sort option")
		}
		if so.GetField() == searchPkg.DocID.String() {
			var cast string
			if schema.ID().SQLType == "uuid" {
				cast = "::text"
			}
			querySoFar.Pagination.OrderBys = append(querySoFar.Pagination.OrderBys, orderByEntry{
				Field: pgsearch.SelectQueryField{
					SelectPath: qualifyColumn(schema.Table, schema.ID().ColumnName, cast),
					FieldType:  postgres.String,
				},
				Descending:  so.GetReversed(),
				SearchAfter: so.GetSearchAfter(),
			})
			continue
		}
		fieldMetadata := queryFields[so.GetField()]
		dbField := fieldMetadata.baseField
		if dbField == nil {
			return errors.Errorf("field %s does not exist in table %s or connected tables", so.GetField(), schema.Table)
		}

		if fieldMetadata.derivedMetadata == nil {
			aggr, distinct := aggregatefunc.GetAggrFuncForV1(so.GetAggregateBy())
			querySoFar.Pagination.OrderBys = append(querySoFar.Pagination.OrderBys, orderByEntry{
				Field:       selectQueryField(so.GetField(), dbField, distinct, aggr, ""),
				Descending:  so.GetReversed(),
				SearchAfter: so.GetSearchAfter(),
			})
		} else {
			var selectField pgsearch.SelectQueryField
			var descending bool
			switch fieldMetadata.derivedMetadata.DerivationType {
			case searchPkg.CountDerivationType:
				selectField = selectQueryField(so.GetField(), dbField, false, aggregatefunc.Count, "")
				descending = so.GetReversed()
			case searchPkg.SimpleReverseSortDerivationType:
				selectField = selectQueryField(so.GetField(), dbField, false, aggregatefunc.Unset, "")
				descending = !so.GetReversed()
			default:
				log.Errorf("Unsupported derived field %s found in query", so.GetField())
				continue
			}

			selectField.DerivedField = true
			querySoFar.Pagination.OrderBys = append(querySoFar.Pagination.OrderBys, orderByEntry{
				Field:      selectField,
				Descending: descending,
			})
		}
	}
	querySoFar.Pagination.Limit = int(pagination.GetLimit())
	querySoFar.Pagination.Offset = int(pagination.GetOffset())
	return nil
}

func applyPaginationForSearchAfter(query *query) error {
	pagination := query.Pagination
	if len(pagination.OrderBys) == 0 {
		return nil
	}
	firstOrderBy := pagination.OrderBys[0]
	if firstOrderBy.SearchAfter == "" {
		return nil
	}
	if query.Where != "" {
		query.Where += " and "
	}
	operand := ">"
	if firstOrderBy.Descending {
		operand = "<"
	}
	query.Where += fmt.Sprintf("%s %s $$", firstOrderBy.Field.SelectPath, operand)
	query.Data = append(query.Data, firstOrderBy.SearchAfter)
	return nil
}
