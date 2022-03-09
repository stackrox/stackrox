package joins

import (
	"fmt"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/postgres/registry"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/search/postgres/mapping"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sliceutils"
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

type joinGeneratorImpl struct {
	// joins holds sql join clauses for source-destination schema pairs.
	joins map[string]map[string]*sqlJoinClauseParts
}

func newJoinGenerator() *joinGeneratorImpl {
	ret := &joinGeneratorImpl{
		joins: make(map[string]map[string]*sqlJoinClauseParts),
	}
	ret.generateJoinsForDBSchema(registry.GetAllRegisteredSchemas())
	return ret
}

func (j joinGeneratorImpl) JoinForCategory(src, dst v1.SearchCategory) ([]string, []string, error) {
	srcSchema, dstSchema := mapping.GetTableFromCategory(src), mapping.GetTableFromCategory(dst)
	if srcSchema == nil {
		return nil, nil, errors.Errorf("no schema registered for search category %q", src)
	}
	if dstSchema == nil {
		return nil, nil, errors.Errorf("no schema registered for search category %q", dst)
	}
	return j.JoinForSchema(srcSchema.Table, dstSchema.Table)
}

func (j joinGeneratorImpl) JoinForSchema(src, dst string) ([]string, []string, error) {
	if src == dst {
		return []string{src}, nil, nil
	}
	dstMap := j.joins[src]
	if dstMap == nil {
		return nil, nil, errors.Errorf("no path registered for source schema %q", src)
	}
	path := dstMap[dst]
	if path == nil {
		return nil, nil, errors.Errorf("no path registered from schema %q to schema %q", src, dst)
	}
	clause := j.joins[src][dst]
	return clause.tables, clause.wheres, nil
}

func (j *joinGeneratorImpl) generateJoinsForDBSchema(schemas map[string]*walker.Schema) {
	// Generate traversal paths from the graph.
	for _, srcSchema := range schemas {
		for _, dstSchema := range schemas {
			if srcSchema == dstSchema {
				continue
			}
			if _, joinExists := j.joins[srcSchema.Table][dstSchema.Table]; joinExists {
				continue
			}
			currJoins := &joins{}
			if search(srcSchema, dstSchema, currJoins, set.NewStringSet()) {
				upsertPathsAndSubPaths(currJoins.toSQLJoinClauseParts(), j.joins)
			}
		}
	}
}

func search(currSchema, dstSchema *walker.Schema, joins *joins, visited set.StringSet) bool {
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
		if !search(parent, dstSchema, joins, visited) {
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
		if !search(child, dstSchema, joins, visited) {
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

func upsertPathsAndSubPaths(sqlJoinParts *sqlJoinClauseParts, joinsMap map[string]map[string]*sqlJoinClauseParts) {
	if sqlJoinParts == nil {
		return
	}

	for i := 0; i < len(sqlJoinParts.tables); i++ {
		if _, ok := joinsMap[sqlJoinParts.tables[i]]; !ok {
			joinsMap[sqlJoinParts.tables[i]] = make(map[string]*sqlJoinClauseParts)
		}

		for j := len(sqlJoinParts.tables) - 1; j > i; j-- {
			src, dst := sqlJoinParts.tables[i], sqlJoinParts.tables[j]
			if _, ok := joinsMap[src][dst]; ok {
				continue
			}

			joinsMap[src][dst] = &sqlJoinClauseParts{
				// There is always one more table than the number of where clauses.
				// For example: ...FROM cluster, namespaces WHERE cluster.id = namespace.cluster_id
				tables: sqlJoinParts.tables[i : j+1],
				wheres: sqlJoinParts.wheres[i:j],
			}
		}
	}
}
