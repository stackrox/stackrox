package postgres

import (
	"sort"
	"testing"

	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stretchr/testify/assert"
)

/*** Helper structs and functions for creation of joinTreeNode tree structure ***/
func columnPair(this, other string) walker.ColumnNamePair {
	return walker.ColumnNamePair{
		ColumnNameInThisSchema:  this,
		ColumnNameInOtherSchema: other,
	}
}

func onColumns(columns ...walker.ColumnNamePair) []walker.ColumnNamePair {
	return columns
}

type joinTableColumns struct {
	table   *joinTreeNode
	columns []walker.ColumnNamePair
}

func toTable(columns []walker.ColumnNamePair, table *joinTreeNode) joinTableColumns {
	return joinTableColumns{
		table:   table,
		columns: columns,
	}
}

func join(joinTables ...joinTableColumns) map[*joinTreeNode][]walker.ColumnNamePair {
	children := make(map[*joinTreeNode][]walker.ColumnNamePair)
	for _, pair := range joinTables {
		children[pair.table] = pair.columns
	}

	return children
}

func table(t string, children map[*joinTreeNode][]walker.ColumnNamePair) *joinTreeNode {
	return &joinTreeNode{
		currNode: &walker.Schema{
			Table: t,
		},
		children: children,
	}
}

/*** Helper structs and functions for creation of test sets ***/
type testSet struct {
	Root            *joinTreeNode
	ReachableFields map[string]searchFieldMetadata
	ExpectedRoot    *joinTreeNode
}

func getTestData() map[string]testSet {
	emptyChildren := make(map[*joinTreeNode][]walker.ColumnNamePair)

	data := map[string]testSet{
		"nil join tree": {
			Root:         nil,
			ExpectedRoot: nil,
		},
		"single table": {
			Root:         table("t1", emptyChildren),
			ExpectedRoot: table("t1", emptyChildren),
		},
		"join on different columns": {
			Root: table("t1",
				join(toTable(onColumns(columnPair("t1_c1", "t2_c1")), table("t2",
					join(toTable(onColumns(columnPair("t2_c2", "t3_c3")), table("t3", emptyChildren))),
				))),
			),
			ExpectedRoot: table("t1",
				join(toTable(onColumns(columnPair("t1_c1", "t2_c1")), table("t2",
					join(toTable(onColumns(columnPair("t2_c2", "t3_c3")), table("t3", emptyChildren))),
				))),
			),
		},
		"join on same column one table to remove": {
			Root: table("t1",
				join(toTable(onColumns(columnPair("t1_c1", "t2_c1")), table("t2",
					join(toTable(onColumns(columnPair("t2_c1", "t3_c1")), table("t3", emptyChildren))),
				))),
			),
			ExpectedRoot: table("t1",
				join(toTable(onColumns(columnPair("t1_c1", "t3_c1")), table("t3", emptyChildren))),
			),
		},
		"join on same column two tables to remove": {
			Root: table("t1",
				join(toTable(onColumns(columnPair("t1_c1", "t2_c1")), table("t2",
					join(toTable(onColumns(columnPair("t2_c1", "t3_c1")), table("t3",
						join(toTable(onColumns(columnPair("t3_c1", "t4_c1")), table("t4", emptyChildren))),
					))),
				))),
			),
			ExpectedRoot: table("t1",
				join(toTable(onColumns(columnPair("t1_c1", "t4_c1")), table("t4", emptyChildren))),
			),
		},
		"one table with same column to remove one table to stay": {
			Root: table("t1",
				join(
					toTable(onColumns(columnPair("t1_c1", "t2_c1")), table("t2",
						join(toTable(onColumns(columnPair("t2_c1", "t3_c1")), table("t3", emptyChildren))),
					)),
					toTable(onColumns(columnPair("t1_c1", "t2stay_c1")), table("t2stay",
						join(toTable(onColumns(columnPair("t2stay_c2", "t4_c1")), table("t4", emptyChildren))),
					)),
				),
			),
			ExpectedRoot: table("t1",
				join(
					toTable(onColumns(columnPair("t1_c1", "t3_c1")), table("t3", emptyChildren)),
					toTable(onColumns(columnPair("t1_c1", "t2stay_c1")), table("t2stay",
						join(toTable(onColumns(columnPair("t2stay_c2", "t4_c1")), table("t4", emptyChildren))),
					)),
				),
			),
		},
		"table will stay when not all columns in child are same": {
			Root: table("t1",
				join(toTable(onColumns(columnPair("t1_c1", "t2_c1")), table("t2",
					join(toTable(onColumns(columnPair("t2_c1", "t3_c1"), columnPair("t2_c2", "t3_c2")), table("t3", emptyChildren))),
				))),
			),
			ExpectedRoot: table("t1",
				join(toTable(onColumns(columnPair("t1_c1", "t2_c1")), table("t2",
					join(toTable(onColumns(columnPair("t2_c1", "t3_c1"), columnPair("t2_c2", "t3_c2")), table("t3", emptyChildren))),
				))),
			),
		},
		"table will stay when not all columns in base are same": {
			Root: table("t1",
				join(toTable(onColumns(columnPair("t1_c1", "t2_c1"), columnPair("t1_c2", "t2_c2")), table("t2",
					join(toTable(onColumns(columnPair("t2_c1", "t3_c1")), table("t3", emptyChildren))),
				))),
			),
			ExpectedRoot: table("t1",
				join(toTable(onColumns(columnPair("t1_c1", "t2_c1"), columnPair("t1_c2", "t2_c2")), table("t2",
					join(toTable(onColumns(columnPair("t2_c1", "t3_c1")), table("t3", emptyChildren))),
				))),
			),
		},
		"join on multiple same columns will remove table": {
			Root: table("t1",
				join(toTable(onColumns(columnPair("t1_c1", "t2_c1"), columnPair("t1_c2", "t2_c2")), table("t2",
					join(toTable(onColumns(columnPair("t2_c1", "t3_c1"), columnPair("t2_c2", "t3_c2")), table("t3", emptyChildren))),
				))),
			),
			ExpectedRoot: table("t1",
				join(toTable(onColumns(columnPair("t1_c1", "t3_c1"), columnPair("t1_c2", "t3_c2")), table("t3", emptyChildren))),
			),
		},
		"required filed will not remove table": {
			ReachableFields: map[string]searchFieldMetadata{
				"test": {
					baseField: &walker.Field{
						Schema: &walker.Schema{
							Table: "t2",
						},
					},
					derivedMetadata: nil,
				},
			},
			Root: table("t1",
				join(toTable(onColumns(columnPair("t1_c1", "t2_c1")), table("t2",
					join(toTable(onColumns(columnPair("t2_c1", "t3_c1")), table("t3", emptyChildren))),
				))),
			),
			ExpectedRoot: table("t1",
				join(toTable(onColumns(columnPair("t1_c1", "t2_c1")), table("t2",
					join(toTable(onColumns(columnPair("t2_c1", "t3_c1")), table("t3", emptyChildren))),
				))),
			),
		},
	}

	return data
}

func TestRemoveUnnecessaryRelations(t *testing.T) {
	t.Parallel()

	testData := getTestData()
	for testName, innerTestRecord := range testData {
		t.Run(testName, func(t *testing.T) {
			innerTestRecord.Root.removeUnnecessaryRelations(innerTestRecord.ReachableFields)

			expectedInnerJoins := innerTestRecord.ExpectedRoot.toInnerJoins()
			innerJoins := innerTestRecord.Root.toInnerJoins()

			// We have to sort before using comparison. Children in joinTreeNode
			// is map where pointers are keys and because of that order changes.
			// It's sufficient to sort by joined tables.
			sort.SliceStable(expectedInnerJoins, func(i, j int) bool {
				return expectedInnerJoins[i].rightTable+expectedInnerJoins[i].leftTable < expectedInnerJoins[j].rightTable+expectedInnerJoins[j].leftTable
			})
			sort.SliceStable(innerJoins, func(i, j int) bool {
				return innerJoins[i].rightTable+innerJoins[i].leftTable < innerJoins[j].rightTable+innerJoins[j].leftTable
			})

			assert.EqualValues(t, expectedInnerJoins, innerJoins)
		})
	}
}
