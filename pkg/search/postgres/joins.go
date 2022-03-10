package postgres

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/utils"
)

type sqlJoinClauseParts struct {
	tables []string
	wheres []string
}

type joinPart struct {
	table      string
	columnName string
}

type join struct {
	lhs *joinPart
	rhs *joinPart
}

type joins []*join

func (j *joins) toSQLJoinClauseParts() *sqlJoinClauseParts {
	if j == nil {
		return nil
	}
	tables, joinsAsStr := make([]string, 0, 2*len(*j)), make([]string, 0, len(*j))
	for _, currJoin := range *j {
		lhs, rhs := currJoin.lhs, currJoin.rhs
		if lhs.table == rhs.table {
			utils.Should(errors.Errorf("LHS table alias %s cannot be the same as RHS table alias %s", lhs.table, rhs.table))
			return nil
		}
		tables = append(tables, rhs.table, lhs.table)
		joinsAsStr = append(joinsAsStr,
			fmt.Sprintf("%s.%s = %s.%s", lhs.table, lhs.columnName, rhs.table, rhs.columnName))
	}

	// Reverse the slice since the recursion constructs slice starting at destination. This is just for improve of sql
	// query as it becomes easy to determine the "path".
	return &sqlJoinClauseParts{
		tables: sliceutils.Reversed(sliceutils.Unique(tables).([]string)).([]string),
		wheres: sliceutils.Reversed(joinsAsStr).([]string),
	}
}

func dedupeAndCombineJoins(src string, joins map[string]*sqlJoinClauseParts) *sqlJoinClauseParts {
	tables := make([]string, 0, len(joins))
	tables = append(tables, src)
	var wheres []string
	visitedTables := set.NewStringSet()
	for _, currJoin := range joins {
		// Not real join. Source is always at 0.
		if len(currJoin.tables) < 2 {
			continue
		}
		for i := 1; i < len(currJoin.tables); i++ {
			if visitedTables.Add(currJoin.tables[i]) {
				continue
			}
			wheres = append(wheres, currJoin.wheres[i-1:i]...)
		}
	}
	return &sqlJoinClauseParts{
		tables: tables,
		wheres: wheres,
	}
}

func getJoin(src *walker.Schema, destinations ...*walker.Schema) ([]string, string) {
	joinsMap := make(map[string]*sqlJoinClauseParts)
	for _, dst := range destinations {
		if src == dst {
			continue
		}
		if _, joinExists := joinsMap[dst.Table]; joinExists {
			continue
		}
		currJoins := &joins{}
		if joinPathRecursive(src, dst, currJoins, set.NewStringSet()) {
			joinsMap[dst.Table] = currJoins.toSQLJoinClauseParts()
		}
	}
	ret := dedupeAndCombineJoins(src.Table, joinsMap)
	return ret.tables, fmt.Sprintf("(%s)", stringutils.JoinNonEmpty(" and ", ret.wheres...))
}

func joinPathRecursive(currSchema, dstSchema *walker.Schema, joins *joins, visited set.StringSet) bool {
	if currSchema == nil || dstSchema == nil {
		return false
	}

	if !visited.Add(currSchema.Table) {
		return false
	}

	if currSchema.Table == dstSchema.Table {
		return true
	}
	if len(currSchema.Parents) == 0 && len(currSchema.Children) == 0 {
		return false
	}

	for _, parent := range currSchema.Parents {
		if !joinPathRecursive(parent, dstSchema, joins, visited) {
			continue
		}

		// Since we are going from child to parent, foreign keys in current schema map to primary keys in parent.
		for _, fk := range currSchema.ForeignKeysReferencesTo(parent.Table) {
			*joins = append(*joins, &join{
				lhs: &joinPart{
					table:      currSchema.Table,
					columnName: fk.ColumnName,
				},
				rhs: &joinPart{
					table:      parent.Table,
					columnName: fk.Reference,
				},
			})
		}
		return true
	}

	for _, child := range currSchema.Children {
		if !joinPathRecursive(child, dstSchema, joins, visited) {
			continue
		}

		// Since we are going from parent to child, primary keys in current schema map to foreign keys in child.
		for _, fk := range child.ForeignKeysReferencesTo(currSchema.Table) {
			*joins = append(*joins, &join{
				lhs: &joinPart{
					table:      currSchema.Table,
					columnName: fk.Reference,
				},
				rhs: &joinPart{
					table:      child.Table,
					columnName: fk.ColumnName,
				},
			})
		}
		return true
	}
	return false
}
