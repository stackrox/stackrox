package gjson

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestColumnNode_Insert(t *testing.T) {
	ct := newColumnTree("test.query", 0)
	ct.rootNode.query = ct.originalQuery
	// should not add a node if it doesn't match the root's query
	assert.False(t, ct.rootNode.Insert(&columnNode{query: "not.matching"}),
		"should not add node with non-matching query")

	newNode := &columnNode{
		value: "abc",
		query: "test",
	}
	assert.True(t, ct.rootNode.Insert(newNode), "should add node with matching query")
	require.Len(t, ct.rootNode.children, 1, "should have exactly 1 child node")
	assert.Equal(t, newNode.value, ct.rootNode.children[0].value, "should have matching value")
	assert.Equal(t, newNode.query, ct.rootNode.children[0].query, "should have matching query")
}

func TestCountDimension(t *testing.T) {

	cases := []struct {
		result            gjson.Result
		expectedDimension int
	}{
		{
			result: gjson.Result{
				Type: gjson.JSON,
				Raw:  `["abc"]`,
			},
			expectedDimension: 1,
		},
		{
			result: gjson.Result{
				Type: gjson.JSON,
				Raw:  `[["abc"]]`,
			},
			expectedDimension: 2,
		},
		{
			result: gjson.Result{
				Type: gjson.JSON,
				Raw:  `[["abc", "def"]]`,
			},
			expectedDimension: 2,
		},
		{
			result: gjson.Result{
				Type: gjson.JSON,
				Raw:  `"abc", "def"`,
			},
			expectedDimension: 0,
		},
		{
			result: gjson.Result{
				Type: gjson.JSON,
				Raw:  `[[[[["abc", "def]]]]]`,
			},
			expectedDimension: 5,
		},
	}
	for _, c := range cases {
		assert.Equalf(t, c.expectedDimension, getDimension(c.result),
			"expected a dimension of %d for result %q", c.expectedDimension, c.result.String())
	}
}

func TestGetValuesAndIndices(t *testing.T) {
	cases := map[string]struct {
		result          gjson.Result
		dimension       int
		expectedValues  []string
		expectedIndices []int
	}{
		"empty result": {
			result: gjson.Result{
				Type: gjson.JSON,
				Raw:  ``,
			},
			expectedValues:  []string{"-"},
			expectedIndices: []int{0},
		},
		"empty array as result": {
			result: gjson.Result{
				Type: gjson.JSON,
				Raw:  `[]`,
			},
			expectedValues:  []string{"-"},
			expectedIndices: []int{0},
		},
		"null type as result": {
			result: gjson.Result{
				Type: gjson.Null,
			},
			expectedValues:  []string{"-"},
			expectedIndices: []int{0},
		},
		"dimension one and one value": {
			result: gjson.Result{
				Type: gjson.JSON,
				Raw:  `["value"]`,
			},
			expectedValues:  []string{"value"},
			expectedIndices: []int{0},
		},
		"multiple values for dimension two": {
			result: gjson.Result{
				Type: gjson.JSON,
				Raw:  `[["valueA","valueB"],["valueC","valueD"]]`,
			},
			dimension:       2,
			expectedValues:  []string{"valueA", "valueB", "valueC", "valueD"},
			expectedIndices: []int{0, 0, 1, 1},
		},
		"multiple values for dimension 6": {
			result: gjson.Result{
				Type: gjson.JSON,
				Raw:  `[[[[[["valueA","valueB"],["valueC","valueD"],["valueE"]]]]]]`,
			},
			dimension:       6,
			expectedValues:  []string{"valueA", "valueB", "valueC", "valueD", "valueE"},
			expectedIndices: []int{0, 0, 1, 1, 2},
		},
		"multiple values and empty values in dimension": {
			result: gjson.Result{
				Type: gjson.JSON,
				Raw:  `[[[["valueA"],[],["valueB", "valueC", "valueD"], []]]`,
			},
			dimension:       4,
			expectedValues:  []string{"valueA", "-", "valueB", "valueC", "valueD", "-"},
			expectedIndices: []int{0, 1, 2, 2, 2, 3},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			values, indices, _ := getValuesAndIndices(c.result, []string{}, -1, []int{}, c.dimension, false)
			assert.Equal(t, c.expectedValues, values)
			assert.Equal(t, c.expectedIndices, indices)
		})
	}
}

func TestColumnNode_CountLeafNodes(t *testing.T) {
	ct := newColumnTree("", 0)
	nodeWith2Leafs := &columnNode{children: []*columnNode{{}, {}}}
	nodeWith5Leafs := &columnNode{children: []*columnNode{{}, {}, {}, {}, {}}}
	ct.rootNode.children = []*columnNode{nodeWith5Leafs, nodeWith2Leafs}

	assert.Equal(t, 7, ct.rootNode.CountLeafNodes(0))
}

func TestColumnNode_GetNodesWithColumnIndex(t *testing.T) {
	ct := newColumnTree("", 0)
	nodeWithColumnIndex2Children := &columnNode{children: []*columnNode{{columnIndex: 2},
		{columnIndex: 2}, {columnIndex: 2}, {columnIndex: 3}}}
	nodeWithoutColumnIndex2Children := &columnNode{children: []*columnNode{{}, {}, {}, {}, {}, {}, {}, {}}}

	ct.rootNode.children = []*columnNode{nodeWithColumnIndex2Children, nodeWithoutColumnIndex2Children}

	assert.Len(t, ct.rootNode.GetNodesWithColumnIndex(2, nil), 3)
}
