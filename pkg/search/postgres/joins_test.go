//go:build sql_integration
// +build sql_integration

package postgres

import (
	"reflect"
	"sort"
	"testing"

	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stretchr/testify/assert"
)

type testSet struct {
	Root            *joinTreeNode
	ReachableFields map[string]searchFieldMetadata
	ExpectedRoot    *joinTreeNode
}

func getTestData() map[string]testSet {
	data := map[string]testSet{
		"nil join tree": {
			Root:         nil,
			ExpectedRoot: nil,
		},
		"single table": {
			Root: &joinTreeNode{
				currNode: &walker.Schema{
					Table: "t1",
				},
				children: make(map[*joinTreeNode][]walker.ColumnNamePair),
			},
			ExpectedRoot: &joinTreeNode{
				currNode: &walker.Schema{
					Table: "t1",
				},
				children: make(map[*joinTreeNode][]walker.ColumnNamePair),
			},
		},
		"join on different columns": {
			Root: &joinTreeNode{
				currNode: &walker.Schema{
					Table: "t1",
				},
				children: map[*joinTreeNode][]walker.ColumnNamePair{
					{
						currNode: &walker.Schema{
							Table: "t2",
						},
						children: map[*joinTreeNode][]walker.ColumnNamePair{
							{
								currNode: &walker.Schema{
									Table: "t3",
								},
								children: make(map[*joinTreeNode][]walker.ColumnNamePair),
							}: {
								{
									ColumnNameInThisSchema:  "t2_c2",
									ColumnNameInOtherSchema: "t3_c3",
								},
							},
						},
					}: {
						{
							ColumnNameInThisSchema:  "t1_c1",
							ColumnNameInOtherSchema: "t2_c1",
						},
					},
				},
			},
			ExpectedRoot: &joinTreeNode{
				currNode: &walker.Schema{
					Table: "t1",
				},
				children: map[*joinTreeNode][]walker.ColumnNamePair{
					{
						currNode: &walker.Schema{
							Table: "t2",
						},
						children: map[*joinTreeNode][]walker.ColumnNamePair{
							{
								currNode: &walker.Schema{
									Table: "t3",
								},
								children: make(map[*joinTreeNode][]walker.ColumnNamePair),
							}: {
								{
									ColumnNameInThisSchema:  "t2_c2",
									ColumnNameInOtherSchema: "t3_c3",
								},
							},
						},
					}: {
						{
							ColumnNameInThisSchema:  "t1_c1",
							ColumnNameInOtherSchema: "t2_c1",
						},
					},
				},
			},
		},
		"join on same column one table to remove": {
			Root: &joinTreeNode{
				currNode: &walker.Schema{
					Table: "t1",
				},
				children: map[*joinTreeNode][]walker.ColumnNamePair{
					{
						currNode: &walker.Schema{
							Table: "t2",
						},
						children: map[*joinTreeNode][]walker.ColumnNamePair{
							{
								currNode: &walker.Schema{
									Table: "t3",
								},
								children: make(map[*joinTreeNode][]walker.ColumnNamePair),
							}: {
								{
									ColumnNameInThisSchema:  "t2_c1",
									ColumnNameInOtherSchema: "t3_c1",
								},
							},
						},
					}: {
						{
							ColumnNameInThisSchema:  "t1_c1",
							ColumnNameInOtherSchema: "t2_c1",
						},
					},
				},
			},
			ExpectedRoot: &joinTreeNode{
				currNode: &walker.Schema{
					Table: "t1",
				},
				children: map[*joinTreeNode][]walker.ColumnNamePair{
					{
						currNode: &walker.Schema{
							Table: "t3",
						},
						children: make(map[*joinTreeNode][]walker.ColumnNamePair),
					}: {
						{
							ColumnNameInThisSchema:  "t1_c1",
							ColumnNameInOtherSchema: "t3_c1",
						},
					},
				},
			},
		},
		"join on same column two tables to remove": {
			Root: &joinTreeNode{
				currNode: &walker.Schema{
					Table: "t1",
				},
				children: map[*joinTreeNode][]walker.ColumnNamePair{
					{
						currNode: &walker.Schema{
							Table: "t2",
						},
						children: map[*joinTreeNode][]walker.ColumnNamePair{
							{
								currNode: &walker.Schema{
									Table: "t3",
								},
								children: map[*joinTreeNode][]walker.ColumnNamePair{
									{
										currNode: &walker.Schema{
											Table: "t4",
										},
										children: make(map[*joinTreeNode][]walker.ColumnNamePair),
									}: {
										{
											ColumnNameInThisSchema:  "t3_c1",
											ColumnNameInOtherSchema: "t4_c1",
										},
									},
								},
							}: {
								{
									ColumnNameInThisSchema:  "t2_c1",
									ColumnNameInOtherSchema: "t3_c1",
								},
							},
						},
					}: {
						{
							ColumnNameInThisSchema:  "t1_c1",
							ColumnNameInOtherSchema: "t2_c1",
						},
					},
				},
			},
			ExpectedRoot: &joinTreeNode{
				currNode: &walker.Schema{
					Table: "t1",
				},
				children: map[*joinTreeNode][]walker.ColumnNamePair{
					{
						currNode: &walker.Schema{
							Table: "t4",
						},
						children: make(map[*joinTreeNode][]walker.ColumnNamePair),
					}: {
						{
							ColumnNameInThisSchema:  "t1_c1",
							ColumnNameInOtherSchema: "t4_c1",
						},
					},
				},
			},
		},
		"one table with same column to remove one table to stay": {
			Root: &joinTreeNode{
				currNode: &walker.Schema{
					Table: "t1",
				},
				children: map[*joinTreeNode][]walker.ColumnNamePair{
					{
						currNode: &walker.Schema{
							Table: "t2",
						},
						children: map[*joinTreeNode][]walker.ColumnNamePair{
							{
								currNode: &walker.Schema{
									Table: "t3",
								},
								children: make(map[*joinTreeNode][]walker.ColumnNamePair),
							}: {
								{
									ColumnNameInThisSchema:  "t2_c1",
									ColumnNameInOtherSchema: "t3_c1",
								},
							},
						},
					}: {
						{
							ColumnNameInThisSchema:  "t1_c1",
							ColumnNameInOtherSchema: "t2_c1",
						},
					},
					{
						currNode: &walker.Schema{
							Table: "t2stay",
						},
						children: map[*joinTreeNode][]walker.ColumnNamePair{
							{
								currNode: &walker.Schema{
									Table: "t4",
								},
								children: make(map[*joinTreeNode][]walker.ColumnNamePair),
							}: {
								{
									ColumnNameInThisSchema:  "t2stay_c2",
									ColumnNameInOtherSchema: "t4_c1",
								},
							},
						},
					}: {
						{
							ColumnNameInThisSchema:  "t1_c1",
							ColumnNameInOtherSchema: "t2stay_c1",
						},
					},
				},
			},
			ExpectedRoot: &joinTreeNode{
				currNode: &walker.Schema{
					Table: "t1",
				},
				children: map[*joinTreeNode][]walker.ColumnNamePair{
					{
						currNode: &walker.Schema{
							Table: "t3",
						},
						children: make(map[*joinTreeNode][]walker.ColumnNamePair),
					}: {
						{
							ColumnNameInThisSchema:  "t1_c1",
							ColumnNameInOtherSchema: "t3_c1",
						},
					},
					{
						currNode: &walker.Schema{
							Table: "t2stay",
						},
						children: map[*joinTreeNode][]walker.ColumnNamePair{
							{
								currNode: &walker.Schema{
									Table: "t4",
								},
								children: make(map[*joinTreeNode][]walker.ColumnNamePair),
							}: {
								{
									ColumnNameInThisSchema:  "t2stay_c2",
									ColumnNameInOtherSchema: "t4_c1",
								},
							},
						},
					}: {
						{
							ColumnNameInThisSchema:  "t1_c1",
							ColumnNameInOtherSchema: "t2stay_c1",
						},
					},
				},
			},
		},
		"table will stay when not all columns in child are same": {
			Root: &joinTreeNode{
				currNode: &walker.Schema{
					Table: "t1",
				},
				children: map[*joinTreeNode][]walker.ColumnNamePair{
					{
						currNode: &walker.Schema{
							Table: "t2",
						},
						children: map[*joinTreeNode][]walker.ColumnNamePair{
							{
								currNode: &walker.Schema{
									Table: "t3",
								},
								children: make(map[*joinTreeNode][]walker.ColumnNamePair),
							}: {
								{
									ColumnNameInThisSchema:  "t2_c1",
									ColumnNameInOtherSchema: "t3_c1",
								},
								{
									ColumnNameInThisSchema:  "t2_c2",
									ColumnNameInOtherSchema: "t3_c2",
								},
							},
						},
					}: {
						{
							ColumnNameInThisSchema:  "t1_c1",
							ColumnNameInOtherSchema: "t2_c1",
						},
					},
				},
			},
			ExpectedRoot: &joinTreeNode{
				currNode: &walker.Schema{
					Table: "t1",
				},
				children: map[*joinTreeNode][]walker.ColumnNamePair{
					{
						currNode: &walker.Schema{
							Table: "t2",
						},
						children: map[*joinTreeNode][]walker.ColumnNamePair{
							{
								currNode: &walker.Schema{
									Table: "t3",
								},
								children: make(map[*joinTreeNode][]walker.ColumnNamePair),
							}: {
								{
									ColumnNameInThisSchema:  "t2_c1",
									ColumnNameInOtherSchema: "t3_c1",
								},
								{
									ColumnNameInThisSchema:  "t2_c2",
									ColumnNameInOtherSchema: "t3_c2",
								},
							},
						},
					}: {
						{
							ColumnNameInThisSchema:  "t1_c1",
							ColumnNameInOtherSchema: "t2_c1",
						},
					},
				},
			},
		},
		"table will stay when not all columns in base are same": {
			Root: &joinTreeNode{
				currNode: &walker.Schema{
					Table: "t1",
				},
				children: map[*joinTreeNode][]walker.ColumnNamePair{
					{
						currNode: &walker.Schema{
							Table: "t2",
						},
						children: map[*joinTreeNode][]walker.ColumnNamePair{
							{
								currNode: &walker.Schema{
									Table: "t3",
								},
								children: make(map[*joinTreeNode][]walker.ColumnNamePair),
							}: {
								{
									ColumnNameInThisSchema:  "t2_c1",
									ColumnNameInOtherSchema: "t3_c1",
								},
							},
						},
					}: {
						{
							ColumnNameInThisSchema:  "t1_c1",
							ColumnNameInOtherSchema: "t2_c1",
						},
						{
							ColumnNameInThisSchema:  "t1_c2",
							ColumnNameInOtherSchema: "t2_c2",
						},
					},
				},
			},
			ExpectedRoot: &joinTreeNode{
				currNode: &walker.Schema{
					Table: "t1",
				},
				children: map[*joinTreeNode][]walker.ColumnNamePair{
					{
						currNode: &walker.Schema{
							Table: "t2",
						},
						children: map[*joinTreeNode][]walker.ColumnNamePair{
							{
								currNode: &walker.Schema{
									Table: "t3",
								},
								children: make(map[*joinTreeNode][]walker.ColumnNamePair),
							}: {
								{
									ColumnNameInThisSchema:  "t2_c1",
									ColumnNameInOtherSchema: "t3_c1",
								},
							},
						},
					}: {
						{
							ColumnNameInThisSchema:  "t1_c1",
							ColumnNameInOtherSchema: "t2_c1",
						},
						{
							ColumnNameInThisSchema:  "t1_c2",
							ColumnNameInOtherSchema: "t2_c2",
						},
					},
				},
			},
		},
		"join on multiple same columns will remove table": {
			Root: &joinTreeNode{
				currNode: &walker.Schema{
					Table: "t1",
				},
				children: map[*joinTreeNode][]walker.ColumnNamePair{
					{
						currNode: &walker.Schema{
							Table: "t2",
						},
						children: map[*joinTreeNode][]walker.ColumnNamePair{
							{
								currNode: &walker.Schema{
									Table: "t3",
								},
								children: make(map[*joinTreeNode][]walker.ColumnNamePair),
							}: {
								{
									ColumnNameInThisSchema:  "t2_c1",
									ColumnNameInOtherSchema: "t3_c1",
								},
								{
									ColumnNameInThisSchema:  "t2_c2",
									ColumnNameInOtherSchema: "t3_c2",
								},
							},
						},
					}: {
						{
							ColumnNameInThisSchema:  "t1_c1",
							ColumnNameInOtherSchema: "t2_c1",
						},
						{
							ColumnNameInThisSchema:  "t1_c2",
							ColumnNameInOtherSchema: "t2_c2",
						},
					},
				},
			},
			ExpectedRoot: &joinTreeNode{
				currNode: &walker.Schema{
					Table: "t1",
				},
				children: map[*joinTreeNode][]walker.ColumnNamePair{
					{
						currNode: &walker.Schema{
							Table: "t3",
						},
						children: make(map[*joinTreeNode][]walker.ColumnNamePair),
					}: {
						{
							ColumnNameInThisSchema:  "t1_c1",
							ColumnNameInOtherSchema: "t3_c1",
						},
						{
							ColumnNameInThisSchema:  "t1_c2",
							ColumnNameInOtherSchema: "t3_c2",
						},
					},
				},
			},
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
			Root: &joinTreeNode{
				currNode: &walker.Schema{
					Table: "t1",
				},
				children: map[*joinTreeNode][]walker.ColumnNamePair{
					{
						currNode: &walker.Schema{
							Table: "t2",
						},
						children: map[*joinTreeNode][]walker.ColumnNamePair{
							{
								currNode: &walker.Schema{
									Table: "t3",
								},
								children: make(map[*joinTreeNode][]walker.ColumnNamePair),
							}: {
								{
									ColumnNameInThisSchema:  "t2_c1",
									ColumnNameInOtherSchema: "t3_c1",
								},
							},
						},
					}: {
						{
							ColumnNameInThisSchema:  "t1_c1",
							ColumnNameInOtherSchema: "t2_c1",
						},
					},
				},
			},
			ExpectedRoot: &joinTreeNode{
				currNode: &walker.Schema{
					Table: "t1",
				},
				children: map[*joinTreeNode][]walker.ColumnNamePair{
					{
						currNode: &walker.Schema{
							Table: "t2",
						},
						children: map[*joinTreeNode][]walker.ColumnNamePair{
							{
								currNode: &walker.Schema{
									Table: "t3",
								},
								children: make(map[*joinTreeNode][]walker.ColumnNamePair),
							}: {
								{
									ColumnNameInThisSchema:  "t2_c1",
									ColumnNameInOtherSchema: "t3_c1",
								},
							},
						},
					}: {
						{
							ColumnNameInThisSchema:  "t1_c1",
							ColumnNameInOtherSchema: "t2_c1",
						},
					},
				},
			},
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

			// We have to sort before using DeepEqual. Otherwise, results could defer.
			// It's sufficient to sort by joined tables.
			sort.SliceStable(expectedInnerJoins, func(i, j int) bool {
				return expectedInnerJoins[i].rightTable+expectedInnerJoins[i].leftTable < expectedInnerJoins[j].rightTable+expectedInnerJoins[j].leftTable
			})
			sort.SliceStable(innerJoins, func(i, j int) bool {
				return innerJoins[i].rightTable+innerJoins[i].leftTable < innerJoins[j].rightTable+innerJoins[j].leftTable
			})

			assert.True(t, reflect.DeepEqual(expectedInnerJoins, innerJoins))
		})
	}
}
