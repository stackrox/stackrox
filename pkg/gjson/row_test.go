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
		query: "test.#.matching",
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
		"multiple values for dimension one": {
			result: gjson.Result{
				Type: gjson.JSON,
				Raw:  `["valueA", "valueB"]`,
			},
			expectedValues:  []string{"valueA", "valueB"},
			expectedIndices: []int{0, 1},
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

	assert.Equal(t, 7, ct.rootNode.countLeafNodes(0))
}

func TestColumnNode_GetNodesWithColumnIndex(t *testing.T) {
	ct := newColumnTree("", 0)
	nodeWithColumnIndex2Children := &columnNode{children: []*columnNode{{columnIndex: 2},
		{columnIndex: 2}, {columnIndex: 2}, {columnIndex: 3}}}
	nodeWithoutColumnIndex2Children := &columnNode{children: []*columnNode{{}, {}, {}, {}, {}, {}, {}, {}}}

	ct.rootNode.children = []*columnNode{nodeWithColumnIndex2Children, nodeWithoutColumnIndex2Children}

	assert.Len(t, ct.rootNode.getNodesWithColumnIndex(2, nil), 3)
}

func TestColumnNode_GetAmountOfUniqueColumnIDsWithinChildren(t *testing.T) {
	cases := map[string]struct {
		node           columnNode
		expectedResult int
	}{
		"only unique column IDs": {
			node:           columnNode{children: []*columnNode{{columnIndex: 1}, {columnIndex: 2}, {columnIndex: 3}}},
			expectedResult: 3,
		},
		"empty children": {
			node:           columnNode{},
			expectedResult: 0,
		},
		"non-unique column IDs": {
			node:           columnNode{children: []*columnNode{{columnIndex: 1}, {columnIndex: 2}, {columnIndex: 2}}},
			expectedResult: 2,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, c.expectedResult, c.node.getAmountOfUniqueColumnIDsWithinChildren())
		})
	}
}

func TestIsRelated(t *testing.T) {
	cases := map[string]struct {
		query         string
		nodes         []*columnNode
		related       bool
		expectedIndex []int
	}{
		"no related nodes": {
			query: "something",
			nodes: []*columnNode{{query: "another.thing"}},
		},
		"multiple related nodes": {
			query: "something.matching.another.thing",
			nodes: []*columnNode{
				{query: "something.matching"},
				{query: "something.matching.another"},
			},
			expectedIndex: []int{0, 1},
			related:       true,
		},
		"mix of related and unrelated nodes": {
			query: "something.matching.another.thing",
			nodes: []*columnNode{
				{query: "something.matching"},
				{query: "something.matching.another"},
				{query: "somewhere.different.query"},
				{query: "somewhere.different"},
			},
			expectedIndex: []int{0, 1},
			related:       true,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			related, index := isRelated(c.query, c.nodes)
			assert.Equal(t, related, c.related)
			assert.Equal(t, c.expectedIndex, index)
		})
	}
}

func TestIsRelatedQuery(t *testing.T) {
	cases := map[string]struct {
		query        string
		relatedQuery string
		related      bool
	}{
		"empty query should be related": {
			related:      true,
			relatedQuery: "something",
		},
		"should be related": {
			query:        "something.matching",
			related:      true,
			relatedQuery: "something.matching.anotherone",
		},
		"should not be related": {
			query:        "something.matching",
			relatedQuery: "anotherhing.matching",
		},
		"should not be related when under the exact same path": {
			query:        "something.matching",
			relatedQuery: "something.othermatch",
		},
		"should not be matching when the queries are equal": {
			query:        "something.matching",
			relatedQuery: "something.matching",
		},
		"should not be matching when no . is used as separator": {
			query:        "something-matching",
			relatedQuery: "something-else-matching",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, c.related, isRelatedQuery(c.relatedQuery, c.query))
		})
	}
}

func TestRowMapper_CreateRows_SimpleHierarchy(t *testing.T) {
	type example struct {
		Names     []string `json:"names"`
		Addresses []string `json:"addresses"`
	}
	type simpleHierarchy struct {
		Result example `json:"result"`
	}

	testJSONObject := &simpleHierarchy{
		Result: example{
			Names:     []string{"Gandalf", "Gollum", "Aragorn", "Bilbo Baggins"},
			Addresses: []string{"Minas Tirith", "Gladden Fields", "Gondor", "Bag End"},
		},
	}
	testExpression := "{result.names,result.addresses}"

	expectedRows := [][]string{
		{"Gandalf", "Minas Tirith"},
		{"Gollum", "Gladden Fields"},
		{"Aragorn", "Gondor"},
		{"Bilbo Baggins", "Bag End"},
	}

	runRowMapperTest(t, testJSONObject, testExpression, expectedRows, []string{})
}

type people struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

type example struct {
	People    []people `json:"people"`
	Franchise string   `json:"franchise"`
}

type deepHierarchy struct {
	Result []example `json:"result"`
}

func TestRowMapper_Create_Rows_DeepHierarchy(t *testing.T) {
	testJSONObject := &deepHierarchy{
		Result: []example{
			{
				Franchise: "LOTR",
				People: []people{
					{
						Name:    "Gandalf",
						Address: "Minas Tirith",
					},
					{
						Name:    "Gollum",
						Address: "Gladden Fields",
					},
					{
						Name:    "Aragorn",
						Address: "Gondor",
					},
					{
						Name:    "Bilbo Baggins",
						Address: "Bag End",
					},
				},
			},
			{
				Franchise: "Harry Potter",
				People: []people{
					{
						Name:    "Harry Potter",
						Address: "Little Whinging",
					},
					{
						Name:    "Ron Weasley",
						Address: "The Burrow",
					},
					{
						Name:    "Hagrid",
						Address: "Hagrid's Hut",
					},
				},
			},
		},
	}

	testExpression := "{result.#.franchise,result.#.people.#.name,result.#.people.#.address}"

	expectedRows := [][]string{
		{"LOTR", "Gandalf", "Minas Tirith"},
		{"LOTR", "Gollum", "Gladden Fields"},
		{"LOTR", "Aragorn", "Gondor"},
		{"LOTR", "Bilbo Baggins", "Bag End"},
		{"Harry Potter", "Harry Potter", "Little Whinging"},
		{"Harry Potter", "Ron Weasley", "The Burrow"},
		{"Harry Potter", "Hagrid", "Hagrid's Hut"},
	}

	runRowMapperTest(t, testJSONObject, testExpression, expectedRows, []string{})
}

func TestRowMapper_CreateRows_DeepHierarchyAndEmptyValues(t *testing.T) {
	testJSONObject := &deepHierarchy{
		Result: []example{
			{
				Franchise: "LOTR",
				People: []people{
					{
						Name:    "Gandalf",
						Address: "Minas Tirith",
					},
					{
						Name:    "Gollum",
						Address: "Gladden Fields",
					},
					{
						Name:    "Aragorn",
						Address: "Gondor",
					},
					{
						Name:    "Bilbo Baggins",
						Address: "Bag End",
					},
					{
						Name: "Sauron",
					},
				},
			},
			{
				Franchise: "Harry Potter",
				People: []people{
					{
						Name:    "Harry Potter",
						Address: "Little Whinging",
					},
					{
						Name:    "Ron Weasley",
						Address: "The Burrow",
					},
					{
						Name:    "Hagrid",
						Address: "Hagrid's Hut",
					},
					{
						Name: "Voldemort",
					},
				},
			},
		},
	}

	testExpression := "{result.#.franchise,result.#.people.#.name,result.#.people.#.address}"

	expectedRows := [][]string{
		{"LOTR", "Gandalf", "Minas Tirith"},
		{"LOTR", "Gollum", "Gladden Fields"},
		{"LOTR", "Aragorn", "Gondor"},
		{"LOTR", "Bilbo Baggins", "Bag End"},
		{"LOTR", "Sauron", "-"},
		{"Harry Potter", "Harry Potter", "Little Whinging"},
		{"Harry Potter", "Ron Weasley", "The Burrow"},
		{"Harry Potter", "Hagrid", "Hagrid's Hut"},
		{"Harry Potter", "Voldemort", "-"},
	}

	runRowMapperTest(t, testJSONObject, testExpression, expectedRows, []string{})
}

func TestRowMapper_CreateRows_DeepHierarchyAndEmptyValuesWithStrictColumns(t *testing.T) {
	testJSONObject := &deepHierarchy{
		Result: []example{
			{
				Franchise: "LOTR",
				People: []people{
					{
						Name:    "Gandalf",
						Address: "Minas Tirith",
					},
					{
						Name: "Gollum",
					},
					{
						Name: "Aragorn",
					},
					{
						Name: "Bilbo Baggins",
					},
					{
						Name: "Sauron",
					},
				},
			},
			{
				Franchise: "Harry Potter",
				People: []people{
					{
						Name:    "Harry Potter",
						Address: "Little Whinging",
					},
					{
						Name: "Ron Weasley",
					},
					{
						Name: "Hagrid",
					},
					{
						Name: "Voldemort",
					},
				},
			},
		},
	}

	testExpression := "{result.#.franchise,result.#.people.#.name,result.#.people.#.address}"

	expectedRows := [][]string{
		{"LOTR", "Gandalf", "Minas Tirith"},
		{"LOTR", "Gollum", "-"},
		{"LOTR", "Aragorn", "-"},
		{"LOTR", "Bilbo Baggins", "-"},
		{"LOTR", "Sauron", "-"},
		{"Harry Potter", "Harry Potter", "Little Whinging"},
		{"Harry Potter", "Ron Weasley", "-"},
		{"Harry Potter", "Hagrid", "-"},
		{"Harry Potter", "Voldemort", "-"},
	}

	expectedRowsWithStrictAddressColumn := [][]string{
		{"LOTR", "Gandalf", "Minas Tirith"},
		{"Harry Potter", "Harry Potter", "Little Whinging"},
	}

	runRowMapperTest(t, testJSONObject, testExpression, expectedRows, []string{})
	runRowMapperTest(t, testJSONObject, testExpression, expectedRowsWithStrictAddressColumn, []string{"result.#.people.#.address"})
}

func runRowMapperTest(t *testing.T, obj interface{}, expression string, expectedRows [][]string, strictColumns []string) {
	mapper, err := NewRowMapper(obj, expression, HideRowsIfColumnNotPopulated(strictColumns))
	require.NoError(t, err)
	rows, err := mapper.CreateRows()
	require.NoError(t, err)
	assert.Equal(t, expectedRows, rows)
}
