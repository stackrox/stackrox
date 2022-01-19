package gjson

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/tidwall/gjson"
)

// RowMapper is responsible for mapping a gjson.Result to a row representation in the form of two-dimensional
// string arrays
type RowMapper struct {
	columnTree *columnTree
}

// NewRowMapper creates a RowMapper which takes a json object and GJSON compatible multi-path JSON expression
// and will retrieve all values from the JSON object and create a row representation in form of a two-dimensional
// string array. Each element within the multi-path JSON expression will be seen as a column value
func NewRowMapper(jsonObj interface{}, multiPathExpression string) (*RowMapper, error) {
	bytes, err := json.Marshal(jsonObj)
	if err != nil {
		return nil, errorhelpers.NewErrInvariantViolation(err.Error())
	}

	result, err := getResultFromBytes(bytes, multiPathExpression)
	if err != nil {
		return nil, err
	}

	ct := constructColumnTree(result, multiPathExpression)

	return &RowMapper{
		columnTree: ct,
	}, nil
}

// CreateRows will firstly retrieve the columns from the gjson.Result, afterwards check if the retrieve columns
// array is a jagged array and then create rows from the columns.
// The creation will also handle expanding singular values of a column to match those of each row.
// Assuming you have constructed the following columns:
// ColA: val1, ColB val2,val3,val4 ColC val5,val6,val7
// will automatically be expanded to
// ColA: val1,val1,val1 ColB val2,val3,val4 ColC val5,val6,val7
// This will only be done if only 1 value is given in the column whereas other columns have multiple
// AND only if the expansion can be done unambiguously.
// If the columns array is jagged, an error will be returned
func (r *RowMapper) CreateRows() ([][]string, error) {
	cols := r.columnTree.CreateColumns()
	if err := isJaggedArray(cols); err != nil {
		return nil, err
	}

	rows := getRowsFromColumns(cols)
	return rows, nil
}

// duplicateElems will duplicate a string element an arbitrary amount and return the slice
func duplicateElems(s string, amount int) []string {
	duplicatedElems := make([]string, 0, amount)
	for i := 0; i < amount; i++ {
		duplicatedElems = append(duplicatedElems, s)
	}
	return duplicatedElems
}

// isJaggedArray will verify whether the given rows array is jagged or not, meaning whether all arrays
// have the same length. It will return an error if the array is jagged.
func isJaggedArray(array [][]string) error {
	if len(array) == 0 {
		return nil
	}

	maxLength := len(array[0])
	for arrIndex, subArray := range array[1:] {
		if maxLength != len(subArray) {
			return jaggedArrayError(maxLength, len(subArray), arrIndex)
		}
	}
	return nil
}

func getResultFromBytes(bytes []byte, jsonPathExpression string) (gjson.Result, error) {
	results := gjson.GetManyBytes(bytes, jsonPathExpression)
	if len(results) != 1 {
		return gjson.Result{}, errorhelpers.NewErrInvariantViolation("expected gjson " +
			"results to be exactly 1")
	}

	return results[0], nil
}

// getRowsFromColumns retrieves all rows from the given columns.
// NOTE: This function relies on the given columns array to not be jagged.
func getRowsFromColumns(columns [][]string) [][]string {
	var rows [][]string
	if len(columns) == 0 {
		return rows
	}

	for colIndex := range columns[0] {
		row := make([]string, 0, len(columns[0]))
		for cellIndex := range columns {
			row = append(row, columns[cellIndex][colIndex])
		}
		rows = append(rows, row)
	}
	return rows
}

// jaggedArrayError helper to create an errorhelpers.ErrInvariantViolation with an explanation about
// a jagged array being found
func jaggedArrayError(maxAmount, violatedAmount, arrayIndex int) error {
	return errorhelpers.NewErrInvariantViolation(fmt.Sprintf("jagged array found: yielded values within "+
		"each array are not matching; expected each array to hold %d elements but found an array with %d elements "+
		"at array index %d", maxAmount, violatedAmount, arrayIndex+1))
}

// columnTree is responsible for providing columns and their values in a tree structure.
// Each node is representing a column value and can be associated with a specific column.
// Each children on the node is related data of another column, i.e. a sub-array.
// The columnTree is constructed by adding nodes as children to related nodes. Nodes are related
// when a) their query is a submatch and b) their relatedIndex matches the parent.
//
// Example:
// Assuming you have a multi-path query such as:
// {result.deployments.#.depName,result.deployments.#.images.#.imgName,result.deployments.#.images.#.components.#.compName,result.deployments.#.images.#.components.#.vulns.#.vulnName}
// which is used against the following JSON object:
// {"result":{"deployments":[{"name":"dep1","images":[{"name":"image1","components":[{"name":"comp11","vulns":[{"name":"cve1"}]},{"name":"comp12"}]},{"name":"image2","components":[{"name":"comp21","vulns":[{"name":"cve1"}]},{"name":"comp22","vulns":[{"name":"cve2"}]}]}]}]}}
//
// The yielded gjson.Result would look like this:
// {"depName":["dep1"],"imgName":[["image1","image2"]],"compName":[[["comp11","comp12"],["comp21","comp22"]]],"vulnName":[[[["cve1"],[]],[["cve1"],["cve2"]]]]}
//
// When constructing the column tree, the query will be sorted for "dimension". A dimension is the depth of arrays
// available per result.
//
// The constructed tree would look like the following:
//                  dep1
//		image1				image2
//	comp11   comp12      comp21   comp22
// cve1          -      cve1          cve2
//
// Each children is representing a related data. Now, when constructing the column, we are aware of
// related data and can expand the column values.
// For example, the "dep1" value needs to be duplicated to be able to create a row successfully.
// The number of all children's leaf nodes is equal to the number of duplication required for the value.
//
// The constructed columns will be the following:
// DEPLOYMENT	IMAGE	COMPONENT	CVE
// Dep1		  	Image1	Comp11		CVE1
// Dep1			Image1	Comp12		-
// Dep1			Image2	Comp21		CVE1
// Dep1			Image2	Comp22		CVE2
// Resulting in the correct duplication of values based on their related data.
type columnTree struct {
	rootNode        *columnNode
	originalQuery   string
	numberOfColumns int
}

// newColumnTree creates a column tree with a root columnNode that has the root property set.
func newColumnTree(query string, numberOfColumns int) *columnTree {
	return &columnTree{
		originalQuery:   query,
		numberOfColumns: numberOfColumns,
		rootNode:        &columnNode{root: true, columnIndex: -1},
	}
}

// columnNode represents a node within a columnTree. Each node represents a value that has been yielded from a
// JSON path query and is associated with a specific columnIndex.
// The dimension of a columnNode specifies the dimension within the result array, the relatedIndex is used to highlight
// the relationship with other data.
type columnNode struct {
	value        string // each value right now is expected to be represented as string
	children     []*columnNode
	dimension    int    // whether the resulted array was one dimensional, two-dimensional etc.
	query        string // original query which resulted in the value. Will be used when inserting values to check whether the subpath matches
	columnIndex  int
	relatedIndex int // this basically is the index in the lower dimension to which the
	// value is related to.
	index int
	root  bool // specified when the node is a root node
}

func constructColumnTree(result gjson.Result, originalQuery string) *columnTree {
	// in multipath queries, the result will be represented as an array.
	res := getQueryResults(result, originalQuery)

	// We need to sort the queries for their dimension, to allow the insertion and relation to be represented correctly.
	// Although we are changing the order, the original column IDs are still retained, so the row creation later on
	// will not be affected.
	sort.SliceStable(res, func(i, j int) bool {
		return res[i].dimension < res[j].dimension
	})

	ct := newColumnTree(originalQuery, len(res))

	for _, r := range res {
		nodes := getColumnNodesPerQuery(r.query, r.result, r.dimension, r.originalIndex)
		for _, node := range nodes {
			ct.rootNode.Insert(node)
		}
	}

	return ct
}

func (ct *columnTree) CreateColumns() [][]string {
	// Get the number of queries. Each query represents a column.
	numberOfQueries := getNumberOfQueries(ct.originalQuery)
	columns := make([][]string, 0, numberOfQueries)
	for columnIndex := 0; columnIndex < numberOfQueries; columnIndex++ {
		// For each query, the query ID == columnID on the node. Retrieve all values for the specific columnID
		// and auto expand, if required, the values already.
		// The values need to be merged based on their index.
		columns = append(columns, ct.createColumnFromColumnNodes(columnIndex))
	}

	return columns
}

type queryResult struct {
	query         string
	result        gjson.Result
	originalIndex int
	dimension     int
}

func getQueryResults(result gjson.Result, originalQuery string) []queryResult {
	q := strings.TrimSuffix(originalQuery, "}")
	q = strings.TrimPrefix(q, "{")
	q = removeModifierExpressionsFromQuery(q)
	queries := strings.Split(q, ",")
	queryIndex := 0

	res := make([]queryResult, 0, len(queries))

	result.ForEach(func(key, value gjson.Result) bool {
		query := queries[queryIndex]
		res = append(res, queryResult{
			query:         query,
			result:        value,
			originalIndex: queryIndex,
			dimension:     getDimensionFromQuery(query),
		})
		queryIndex++
		return true
	})
	return res
}

func getNumberOfQueries(query string) int {
	q := strings.TrimSuffix(query, "}")
	q = strings.TrimPrefix(q, "{")
	q = removeModifierExpressionsFromQuery(q)
	queries := strings.Split(q, ",")
	return len(queries)
}

func getDimensionFromQuery(query string) int {
	return strings.Count(query, "#")
}

// isRelatedQuery checks whether relatedQuery is a relatedQuery to query
func isRelatedQuery(relatedQuery, query string) bool {
	// related queries are substrings and not equal. If they are equal, return false
	if relatedQuery == query {
		return false
	}
	// we need to trim everything after the last "." for comparison, since the access object will certainly be different
	if idx := strings.LastIndex(query, "."); idx != -1 {
		query = query[:idx]
	}

	if idx := strings.LastIndex(relatedQuery, "."); idx != -1 {
		relatedQuery = relatedQuery[:idx]
	}

	if relatedQuery == query {
		return false
	}

	for _, split := range strings.Split(query, ".") {
		if strings.Contains(relatedQuery, split) {
			return true
		}
	}

	return false
}

func getColumnNodesPerQuery(query string, result gjson.Result, dimension int, columnIndex int) []*columnNode {
	if result.Type == gjson.Null {
		return nil
	}
	if !result.IsArray() {
		return []*columnNode{{
			value:       result.String(),
			children:    nil,
			dimension:   dimension,
			columnIndex: columnIndex,
			query:       query,
		}}
	}

	values, indices, _ := getValuesAndIndices(result, []string{}, -1, []int{}, dimension, false)

	nodes := make([]*columnNode, 0, len(values))

	for i, v := range values {
		nodes = append(nodes, &columnNode{
			value:        v,
			children:     nil,
			dimension:    dimension,
			query:        query,
			columnIndex:  columnIndex,
			relatedIndex: indices[i],
			index:        i,
		})
	}

	return nodes
}

// getValuesAndIndices retrieves the values from the current result as well as the indices they relate to in the
// lower dimension.
// The values of lastIndex and count have to be false when starting the recursion.
func getValuesAndIndices(value gjson.Result, values []string, lastIndex int, indicesInLowerDimension []int, dimension int, count bool) ([]string, []int, int) {
	if value.String() == "" || value.Raw == "[]" || value.Type == gjson.Null {
		if dimension <= 1 {
			// Special case: If the dimension is 0, we always count. Otherwise, this would lead to lastIndex=-1.
			lastIndex++
		}
		return append(values, "-"), append(indicesInLowerDimension, lastIndex), lastIndex
	}

	if !value.IsArray() {
		return append(values, value.String()), append(indicesInLowerDimension, lastIndex), lastIndex
	}
	if value.IsArray() {
		arr := value.Array()
		for _, res := range arr {
			if res.IsArray() && count && getDimension(res) == 1 {
				lastIndex++
			}
			// Start counting when the value is equal to the dimension we are getting values for.
			// Count the index of associated values in lower hierarchy.
			if currentDimension := getDimension(value); currentDimension == dimension && !count {
				count = true
				// Special case: Since the lastIndex is starting with -1, need to count for dimension <=2, otherwise
				// the result will be lastIndex=-1, which would be invalid
				if currentDimension <= 2 {
					lastIndex++
				}
			} else if dimension <= 1 {
				// Special case: If the dimension is 0, we always count. Otherwise, this would lead to lastIndex=-1.
				count = true
				lastIndex++
			}

			// TODO: Potentially need a special case here, where we are counting for dimension 1 each time.

			// Recursively get the values and indices for each array element. The lastIndex will be reused, so get the
			// correct offset for each yielded values.
			values, indicesInLowerDimension, lastIndex = getValuesAndIndices(res, values, lastIndex, indicesInLowerDimension, dimension, count)
		}
	}

	return values, indicesInLowerDimension, lastIndex
}

func (n *columnNode) isInTheSameDimensionAsChildren(indexOfChildren []int, dimension int) bool {
	for _, i := range indexOfChildren {
		if n.children[i].dimension >= dimension {
			return true
		}
	}
	return false
}

// Insert will insert the given columnNode into the existing node. This can be either called on the root node or any
// columnNode.
// The node will be inserted when the queries are related and the inserted node's relatedIndex is equal to the existing nodes index.
// Special case: The node will always be added to the root node, no matter of the index.
// When the node is inserted successfully, it will return true, otherwise false.
func (n *columnNode) Insert(node *columnNode) bool {
	// Add the node only if it is related
	if isRelatedQuery(node.query, n.query) {
		// Adding the node to a children first if it is related to one. If not, we add it to the current node.
		if isRelatedToChildren, indexOfChildren := isRelatedOfChildren(node.query, n.children); isRelatedToChildren &&
			n.isInTheSameDimensionAsChildren(indexOfChildren, n.dimension) {
			res := false
			for _, ind := range indexOfChildren {
				res = n.children[ind].Insert(node)
				if res {
					return res
				}
			}
			return res
		}
		// If we are adding to the root node, indices do not matter.
		if n.index == node.relatedIndex || n.root {
			n.children = append(n.children, node)
			return true
		}

	}

	return false
}

func (ct *columnTree) createColumnFromColumnNodes(columnIndex int) []string {
	nodes := ct.rootNode.GetNodesWithColumnIndex(columnIndex, []*columnNode{})

	// Sort the nodes by index. This is important to be able to construct the column.
	sort.SliceStable(nodes, func(i, j int) bool {
		return nodes[i].index < nodes[j].index
	})

	var column []string
	for _, node := range nodes {
		amount := node.CountLeafNodes(0)

		// Special case: The node has _all_ other column nodes as children. This means, we need to duplicate the
		// value not by the amount of leaf nodes, but by the amount of leaf nodes / number of columns.
		if node.getAmountOfUniqueColumnIDsWithinChildren() == ct.numberOfColumns-1 {
			amount = amount / (ct.numberOfColumns - 1)
		}
		column = append(column, duplicateElems(node.value, amount)...)
	}

	return column
}

// GetNodesWithColumnIndex returns recursively all nodes and children nodes that have a matching column index.
func (n *columnNode) GetNodesWithColumnIndex(columnIndex int, nodes []*columnNode) []*columnNode {
	if n.columnIndex == columnIndex {
		nodes = append(nodes, n)
	}
	for _, child := range n.children {
		nodes = child.GetNodesWithColumnIndex(columnIndex, nodes)
	}

	return nodes
}

// CountLeafNodes counts the leaf nodes associated with the current node and its children recursively.
func (n *columnNode) CountLeafNodes(offset int) int {
	if len(n.children) == 0 {
		return offset + 1
	}

	for _, child := range n.children {
		offset = child.CountLeafNodes(offset)
	}

	return offset
}

func (n *columnNode) GetAmountOfDifferentColumnsAsChildren(visited map[int]struct{}) map[int]struct{} {
	if _, exists := visited[n.columnIndex]; !exists {
		visited[n.columnIndex] = struct{}{}
	}

	for _, child := range n.children {
		visited = child.GetAmountOfDifferentColumnsAsChildren(visited)
	}

	return visited
}

func (n *columnNode) getAmountOfUniqueColumnIDsWithinChildren() int {
	visited := map[int]struct{}{}
	for _, child := range n.children {
		if _, exists := visited[child.columnIndex]; !exists {
			visited[child.columnIndex] = struct{}{}
		}
	}
	return len(visited)
}

func isRelatedOfChildren(query string, nodes []*columnNode) (bool, []int) {
	relatedChildren := []int{}
	match := false
	for index, node := range nodes {
		if isRelatedQuery(query, node.query) {
			relatedChildren = append(relatedChildren, index)
			match = true
		}
	}
	return match, relatedChildren
}

func getDimension(result gjson.Result) int {
	return countDimension(result, 0)
}

func countDimension(result gjson.Result, offset int) int {
	if !result.IsArray() || result.Type == gjson.Null {
		return offset
	}

	offset++

	result.ForEach(func(key, value gjson.Result) bool {
		offset = countDimension(value, offset)
		return false
	})

	return offset
}

func removeModifierExpressionsFromQuery(query string) string {
	regs := ModifiersRegexp()
	for _, r := range regs {
		query = r.ReplaceAllString(query, "")
	}
	return query
}
