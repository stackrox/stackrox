package walker

import (
	"fmt"
	"log"
	"strings"

	"github.com/stackrox/rox/pkg/pointers"
)

// RenderGetQueryWithRollup renders a get query that returns the full object without depending on the serialized field.
func (s *Schema) RenderGetQueryWithRollup() string {
	return s.getWithRollupQueryHelper(nil, pointers.Int(0))
}

func getTableAliasAndIncrement(idxsUsedSoFar *int) string {
	alias := fmt.Sprintf("table%d", *idxsUsedSoFar)
	*idxsUsedSoFar++
	return alias
}

func getJoinClauseAlias(idx int) string {
	return fmt.Sprintf("join%d", idx)
}

type parentMatcherDesc struct {
	parentTableAlias   string
	columnNameInParent string
	columnNameInChild  string
}

func (s *Schema) getWithRollupQueryHelper(parentMatchingClauses []parentMatcherDesc, idxsUsedSoFar *int) string {
	isRoot := len(parentMatchingClauses) == 0
	thisTableAlias := getTableAliasAndIncrement(idxsUsedSoFar)
	var selectArgs []string
	var childRollupQueries []string

	newParentMatchingClauses := append([]parentMatcherDesc{}, parentMatchingClauses...)
	for _, field := range s.Fields {
		if isRoot {
			selectArgs = append(selectArgs, fmt.Sprintf("%s.%s as %s", thisTableAlias, field.ColumnName, field.ColumnName))

		} else {
			selectArgs = append(selectArgs, fmt.Sprintf("'%s'", field.ColumnName), fmt.Sprintf("%s.%s", thisTableAlias, field.ColumnName))
		}
		if field.Options.PrimaryKey {
			newParentMatchingClauses = append(newParentMatchingClauses,
				parentMatcherDesc{
					parentTableAlias:   thisTableAlias,
					columnNameInParent: field.ColumnName,
					columnNameInChild:  parentify(s.Table, field.ColumnName),
				})
		}
	}
	for i, child := range s.Children {
		childResultAlias := getJoinClauseAlias(i)
		if isRoot {
			selectArgs = append(selectArgs, fmt.Sprintf("to_json(%s)->'array' as %s", childResultAlias, childResultAlias))
		} else {
			selectArgs = append(selectArgs, fmt.Sprintf("'%s'", childResultAlias), fmt.Sprintf("to_json(%s)->'array'", childResultAlias))
		}
		childRollupQueries = append(childRollupQueries, child.getWithRollupQueryHelper(newParentMatchingClauses, idxsUsedSoFar))
	}
	var entireQuery strings.Builder
	// Unfortunately, postgres has a limit of 100 arguments that can be passed to a function.
	// This means that json_build_object won't work for objects that have 50 or more fields at the root level.
	// So we use the row_to_json construct as an alternative (derived from https://stackoverflow.com/a/41845805/3690207).
	// However, row_to_json does NOT work for queries that might return multiple rows.
	// So we can't use it for any child queries.
	// TODO: there is probably a way to make this happen.
	if isRoot {
		entireQuery.WriteString(fmt.Sprintf("select row_to_json((select record from (select %s from %s %s", strings.Join(selectArgs, ", "), s.Table, thisTableAlias))
	} else {
		if len(selectArgs) > 100 {
			log.Panicf("Cannot generate rollup for table %s (path: %+v): it has more than 50 fields", s.Table, parentMatchingClauses)
		}
		entireQuery.WriteString(fmt.Sprintf("select json_build_object(%s) from %s %s", strings.Join(selectArgs, ", "), s.Table, thisTableAlias))
	}

	for i, childRollupQuery := range childRollupQueries {
		childResultAlias := getJoinClauseAlias(i)
		entireQuery.WriteString(fmt.Sprintf(" left join lateral (select array(%s)) %s on true", childRollupQuery, childResultAlias))
	}
	var whereClauseEntries []string
	if isRoot {
		// This is the parent, so we should put the "where primary key matches" clause
		primaryKeys := s.LocalPrimaryKeys()
		if len(primaryKeys) == 0 {
			log.Panicf("No primary keys in table %+v", s)
		}
		for i, pk := range primaryKeys {
			whereClauseEntries = append(whereClauseEntries, fmt.Sprintf("%s.%s = $%d", thisTableAlias, pk.ColumnName, i+1))
		}
	} else {

		// This is a child table, so just query via parent.
		for _, clause := range parentMatchingClauses {
			whereClauseEntries = append(whereClauseEntries, fmt.Sprintf("%s.%s = %s.%s", clause.parentTableAlias, clause.columnNameInParent, thisTableAlias, clause.columnNameInChild))
		}
	}
	entireQuery.WriteString(fmt.Sprintf(" where (%s)", strings.Join(whereClauseEntries, " and ")))
	// This closes the parentheses opened in the row_to_json above.
	if isRoot {
		entireQuery.WriteString(") record ))")
	}
	return entireQuery.String()
}
