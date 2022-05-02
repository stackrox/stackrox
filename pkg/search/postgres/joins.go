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

func (j *joins) withSchemaRelationship(currSchema *walker.Schema, rel walker.SchemaRelationship) joins {
	newJoins := make(joins, len(*j))
	copy(newJoins, *j)
	for _, mappedColumnName := range rel.MappedColumnNames {
		newJoins = append(newJoins, &join{
			lhs: &joinPart{
				table:      currSchema.Table,
				columnName: mappedColumnName.ColumnNameInThisSchema,
			},
			rhs: &joinPart{
				table:      rel.OtherSchema.Table,
				columnName: mappedColumnName.ColumnNameInOtherSchema,
			},
		})
	}
	return newJoins
}

func (j *joins) toSQLJoinClauseParts() *sqlJoinClauseParts {
	if j == nil {
		return nil
	}
	tables := make([]string, 0, 2*len(*j))
	joinsAsStr := make([]string, 0, len(*j))
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

// getJoins returns join clauses to join src to destinations as a map keyed on destination table name.
func getJoins(src *walker.Schema, destinations ...*walker.Schema) ([]string, map[string]string) {
	joinMap := make(map[string]*sqlJoinClauseParts)
	for _, dst := range destinations {
		if src == dst {
			continue
		}
		if _, joinExists := joinMap[dst.Table]; joinExists {
			continue
		}
		joinsToDest, found := findJoins(src, dst)
		if found {
			joinMap[dst.Table] = joinsToDest.toSQLJoinClauseParts()
		}
	}

	tables := set.NewStringSet(src.Table)
	joinStrMap := make(map[string]string)
	for dst, currJoin := range joinMap {
		tables.AddAll(currJoin.tables...)
		joinStrMap[dst] = stringutils.JoinNonEmpty(" and ", currJoin.wheres...)
	}

	return tables.AsSortedSlice(func(i, j string) bool { return i < j }), joinStrMap
}

type bfsQueueElem struct {
	joinsSoFar joins
	currSchema *walker.Schema
}

func findJoins(srcSchema, dstSchema *walker.Schema) (joins, bool) {
	visited := set.NewStringSet()

	queue := []bfsQueueElem{{joinsSoFar: joins{}, currSchema: srcSchema}}
	// We want to traverse shortest length from current schema to the other, so do it via BFS.
	for len(queue) > 0 {
		currElem := queue[0]
		queue = queue[1:]

		if !visited.Add(currElem.currSchema.Table) {
			continue
		}

		if currElem.currSchema == dstSchema {
			return currElem.joinsSoFar, true
		}

		for _, rel := range currElem.currSchema.AllRelationships() {
			newElem := bfsQueueElem{
				joinsSoFar: currElem.joinsSoFar.withSchemaRelationship(currElem.currSchema, rel),
				currSchema: rel.OtherSchema,
			}
			queue = append(queue, newElem)
		}
	}

	return joins{}, false
}
