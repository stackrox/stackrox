package gjson

import (
	"encoding/json"
	"fmt"

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
