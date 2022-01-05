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
	// Within the matrix, each array is representing the amount of values for each column per query, this is
	// the column position.
	matrix [][]int
	result gjson.Result
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

	// Testing it here since its easier
	constructColumnTree(result, multiPathExpression)

	matrix := createRowMapperMatrix(result)

	if err := validateMatrix(matrix); err != nil {
		return nil, err
	}

	return &RowMapper{
		matrix: matrix,
		result: result,
	}, nil
}

// getNumOfElementsForCol gets the number of elements for a column at the specific position
func (r *RowMapper) getNumOfElementsForCol(colPosition int) int {
	for _, data := range r.matrix {
		if data[colPosition] != 1 {
			return data[colPosition]
		}
	}
	return 1
}

// createColumns retrieves the column values from the gjson.Result in form of a two-dimensional string array
func (r *RowMapper) createColumns() [][]string {
	var result [][]string
	if r.matrix == nil {
		return result
	}
	colID := 0

	r.result.ForEach(func(key, value gjson.Result) bool {
		// Only try to retrieve string values from the result if it is not an empty array.
		var row []string
		if value.Type != gjson.Null {
			row = getStringValuesFromNestedArrays(value, []string{})
		}
		postProcessedRow := r.expandColumn(row, colID)
		result = append(result, postProcessedRow)
		colID++
		return true
	})
	return result
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
	cols := r.createColumns()
	if err := isJaggedArray(cols); err != nil {
		return nil, err
	}

	rows := getRowsFromColumns(cols)
	return rows, nil
}

// expandColumn will duplicate column values for each column position for a specific column.
// Sometimes, data is expected to be the same for a number of values, such as a name for an image etc.
// These static values need to be duplicated, since we json path expressions do not yield it multiple times,
// which would result in a jagged array.
// To overcome this, all columns which are only yielding one value for a specific column position, and other
// columns are yielding multiple values for this position, the value will be duplicated to match other column's
// values
func (r *RowMapper) expandColumn(col []string, colID int) []string {
	values := r.matrix[colID]
	offset := 0
	for colPosition, value := range values {
		expectedValue := r.getNumOfElementsForCol(colPosition)
		if expectedValue != value {
			// add one less, since we have the original element already available, it will not be overridden
			elemsToAdd := duplicateElems(col[offset], expectedValue-1)
			col = insertIntoStringSlice(col, offset, elemsToAdd...)
			offset += len(elemsToAdd)
		}
	}
	return col
}

// createRowMapperMatrix will initialize the matrix from a gjson.Result. Each array is representing the amount of
// values yielded per result. Since multi-path expression can yield nested arrays, each amount of yielded values will
// be added.
func createRowMapperMatrix(result gjson.Result) [][]int {
	var matrix [][]int
	result.ForEach(func(key, value gjson.Result) bool {
		// Will check here the length of each value's array, potentially we can have nested arrays here, hence the 2d
		// array
		matrix = append(matrix, getNumOfValuesForCol(value))
		return true
	})
	return matrix
}

// getNumOfValuesForCol returns the number of values which each column position holds, the gjson.Result representing
// a column
func getNumOfValuesForCol(value gjson.Result) []int {
	// for now, only supporting two-dimensional arrays within results, not an arbitrary amount of nested arrays
	var matrix []int
	value.ForEach(func(key, value gjson.Result) bool {
		if !value.IsArray() {
			matrix = append(matrix, 1)
			return true
		}
		amount := 0
		value.ForEach(func(key, value gjson.Result) bool {
			amount++
			return true
		})
		matrix = append(matrix, amount)
		return true
	})
	return matrix
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

// insertIntoStringSlice will insert given elements at the index and return the
// changed array
func insertIntoStringSlice(s []string, index int, elems ...string) []string {
	return append(s[:index], append(elems, s[index:]...)...)
}

// jaggedArrayError helper to create an errorhelpers.ErrInvariantViolation with an explanation about
// a jagged array being found
func jaggedArrayError(maxAmount, violatedAmount, arrayIndex int) error {
	return errorhelpers.NewErrInvariantViolation(fmt.Sprintf("jagged array found: yielded values within "+
		"each array are not matching; expected each array to hold %d elements but found an array with %d elements "+
		"at array index %d", maxAmount, violatedAmount, arrayIndex+1))
}

func validateMatrix(matrix [][]int) error {
	if len(matrix) == 0 {
		return nil
	}

	maxLength := len(matrix[0])
	for arrIndex, subArray := range matrix[1:] {
		if maxLength != len(subArray) {
			return jaggedArrayError(maxLength, len(subArray), arrIndex)
		}
	}
	return nil
}


/*
Problem: Currently, we only expand values within columns and are not taking into account potential relationships and hierarchy
between them. This will lead to a failure in expanding the columns when having to expand them in different amounts based
on the hierarchy.

Idea: We need to introduce a way to check the hierarchy first, then we can also take into account potential relationships
between data.
Hierachy in the JSON path world would be the number of dimension of the result array. We structure the results as a tree,
where the depth is equivalent to the dimension of the yielded array, i.e.
0 -> root
1 -> one dimensional array
2 -> two dimensional array
3 -> three dimensional array
and so on.

This will also allow us to fix the relationship between data. The children nodes will represent that there is a relationship
between the results.

Columns would now be represented by a tree, instead of a two dimensional string array.

The challenge is to fill the tree correctly, this includes the following:
- Finding the correct hierarchy by using the dimension of the result
- Checking whether the data has any relation to previously inserted nodes

The relationship is, right now and also within JSON, done by having JSON objects be subitems of another object. This
is also reflected within the path "data.object.item". It is safe to say that an object has a relation when it is a subpath
of a different JSON path query.

The expansion of the values will be handled by traversing the tree nodes and duplicating the related columns by the
amount of children branches, through all dimensions. If only one branch exists, no expansion needs to take place. If
there are multiple branches, the amount of branches will be summed up (i.e. 2 branches for dimension 2, 5 branches in
dimension 3 results in the expansion of 7 for the node).
 */

type columnTree struct {
	rootNode *columnNode
	originalQuery string // not sure we need this right here
}

type columnNode struct {
	value string // each value right now is expected to be represented as string
	children []*columnNode
	dimension int // whether the resulted array was one dimensional, two-dimensional etc.
	query string // original query which resulted in the value. Will be used when inserting values to check whether the subpath matches
	relatedIndex int // not sure how to model this, but this basically is the index in the lower dimension to which the
	// value is related to.
	index int
}

func constructColumnTree(result gjson.Result, originalQuery string) *columnTree {
	// in multipath queries, the result will be represented as an array.
	res := getQueryResults(result, originalQuery)

	// sort results by dimension, from lowest to highest
	sort.SliceStable(res, func(i, j int) bool {
		return res[i].dimension < res[j].dimension
	})

	ct := &columnTree{originalQuery: originalQuery, rootNode: &columnNode{}}

	for _, r := range res {
		nodes := getColumnNodesPerQuery(r.query, r.result, r.dimension)
		for _, node := range nodes {
			ct.rootNode.Insert(node)
		}
	}

	return ct
}

type queryResult struct {
	query string
	result gjson.Result
	dimension int
}

func getQueryResults(result gjson.Result, originalQuery string) []queryResult {
	q := strings.TrimSuffix(originalQuery, "}")
	q = strings.TrimPrefix( q, "{")
	queries := strings.Split(q, ",")
	queryIndex := 0

	res := make([]queryResult, 0, len(queries))

	result.ForEach(func(key, value gjson.Result) bool {
		query := queries[queryIndex]
		res = append(res, queryResult{
			query:     query,
			result:    value,
			dimension: getDimensionFromQuery(query),
		})
		queryIndex++
		return true
	})
	return res
}

func getDimensionFromQuery(query string) int {
	return strings.Count(query, "#")
}
// isRelatedQuery checks whether relatedQuery is a relatedQuery to query
func isRelatedQuery(relatedQuery, query string) bool {
	// related queries are substrings and not equal. If they are equal, return false
	if relatedQuery == query  {
		return false
	}
	// we need to trim everything after the last "." for comparison, since the access object will certainly be different
	if idx := strings.LastIndex(query, "."); idx != - 1 {
		query = query[:idx]
	}
	return strings.Contains(relatedQuery, query)
}

func getColumnNodesPerQuery(query string, result gjson.Result, dimension int) []*columnNode {
	if !result.IsArray() {
		return []*columnNode{{
			value:     result.String(),
			children:  nil,
			dimension: dimension,
			query:     query,

		}}
	}

	values, indices := getValuesAndIndices(result, []string{}, -1, []int{}, dimension, false)

	nodes := make([]*columnNode, 0, len(values))

	for i, v := range values {
		nodes = append(nodes, &columnNode{
			value:     v,
			children:  nil,
			dimension: dimension,
			query:     query,
			relatedIndex: indices[i],
			index: i,
		})
	}

	return nodes
}

func getValuesAndIndices(value gjson.Result, values []string, lastIndex int, indicesInLowerDimension []int, dimension int, count bool) ([]string, []int) {
	if value.String() == "" || value.Type == gjson.Null {
		return append(values, "-"), append(indicesInLowerDimension, lastIndex)
	}

	if !value.IsArray() {
		return append(values, value.String()), append(indicesInLowerDimension, lastIndex)
	}

	// This still doesn't work. I cannot properly count the index of the lower level dimension to be like "hey, this is for 0, this is for 1 etc.".
	// We need to know "OK, these values are from the lower dimension, need to get their position to determine FOR which index they are referred to.

// [[["comp11","comp12"],["comp21","comp22"]]]
//       0         1         2         3

// [[[["cve1"],[]],[["cve1"],["cve2"]]]]
//        0     1      2         3
// Issue is
/*
	[["cve1"],[]] this is one array -> elem 0 points to "comp11" 0, empty is empty for comp12 1
	[["cve1"],["cve2"]] this is one array -> elem 0 points to comp21 (which is 2). elem 1 points to comp22 (which is 3).

So the goal would be the following:
- We know at which point the value we are having within our array is the array for the dimension (this might be a two parter)
- We are now marking that we need to count the AMOUNT OF OBJECTS yielded within the array

(recursive)
- In the next iteration, we are doing the following:
     if the value is just a single value we return the index + value
	 if the value is more than a single value AND the value is an array, we need to count the amount of values we are having but ONLY if the counter
     has been set already. Afterwards, we need to return the value back for the invocation
(end recursive)

- We are back in the loop. If the marker was set for counting, we use the yielded amount and add it up to the current index
- Within the next iteration, the game begins again, but with an offset yielded from the previous invocation


- We detect the dimension (lower one) correctly

- We detect the array correctly IF the value itself is not a single array, i.e. [[["comp11","comp12"],["comp21","comp22"]]] is detectable,
  [[[["cve1"],[]],[["cve1"],["cve2"]]]] is not.

- We count too many times w.r.t to arrays. Now, is this because we started at the wrong dimension? Or is this because theres another flaw in the logic?
 */
	if value.IsArray() {
		arr := value.Array()
		for _, res := range arr {
			// There is still an issue here with comp11 / cve1: We count one too many times, but weirdly only for this component and not for the others.
			// This is not correct. We need to get the index back from which we are finished with the element for subsequent invocations.
			// Only issue is to deal with the offset properly.
			if res.IsArray() && count {
				lastIndex++
			}

			// Find out whether we are in the n-1 dimension
			if currentDimension:= getDimension(res); currentDimension == dimension - 1 {
				count = true
				// special case for dimension 1 / 2: need to increase the last index here.
				if currentDimension < 2 {
					lastIndex++
				}
			}

			values, indicesInLowerDimension = getValuesAndIndices(res, values, lastIndex, indicesInLowerDimension, dimension, count)
		}
	}

	return values, indicesInLowerDimension
}

func (n *columnNode) Insert(node *columnNode) bool {
	// Add the node only if it is related
	if isRelatedQuery(node.query, n.query) {
		// Adding the node to a children first if it is related to one. If not, we add it to the current node.
		if isRelatedToChildren, indexOfChildren := isRelatedOfChildren(node.query, node.relatedIndex, n.children); isRelatedToChildren {
			res := false
			for _, ind := range indexOfChildren {
				res = n.children[ind].Insert(node)
			}
			return res
		}
		// Do we need to check whether the related index matches? If this is the case (related, not related to
		// any children BUT a different index is expected, this is potentially an error.
		if n.index == node.relatedIndex {
			n.children = append(n.children, node)
		}
		return true
	}

	return false
}

func isRelatedOfChildren(query string, relatedIndex int, nodes []*columnNode) (bool, []int) {
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

func getDimensionOfResult(result gjson.Result) int {
	count := strings.Count(result.String(), "[")
	return count
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
