package printer

import (
	"encoding/json"
	"fmt"

	"github.com/tidwall/gjson"
)

// JaggedArrayError creates a standardized error for jagged arrays when yielding values via gjson
type JaggedArrayError struct {
	maxAmount      int
	violatedAmount int
	rowIndex       int
}

func (j JaggedArrayError) Error() string {
	return fmt.Sprintf("jagged array found: yielded values within each array are not matching. "+
		"Expected each array to hold %d elements but found an array with %d at row index %d",
		j.maxAmount, j.violatedAmount, j.rowIndex+1)
}

// getRowsFromObject will retrieve all rows from the json object. It will use gjson and the
// rowExpression to retrieve all row values and return them as an array.
// It will return an error if either the given json object could not be passed to json.Marshal
// or the retrieved rows array is jagged.
func getRowsFromObject(jsonObj interface{}, rowExpression string) ([][]string, error) {
	jsonBytes, err := json.Marshal(jsonObj)
	if err != nil {
		return nil, err
	}

	result := gjson.GetManyBytes(jsonBytes, rowExpression)
	columns := gjsonResultsToColumns(result)
	if err := isJaggedArray(columns); err != nil {
		return nil, err
	}
	rows := getRowsFromColumns(columns)
	return rows, nil
}

// isJaggedArray will verify whether the given rows array is jagged or not, meaning whether all arrays
// have the same length. It will return an error if the array is jagged.
func isJaggedArray(rows [][]string) error {
	if len(rows) == 0 {
		return nil
	}

	maxRowLength := len(rows[0])
	for rowIndex, row := range rows[1:] {
		if maxRowLength != len(row) {
			return JaggedArrayError{maxAmount: maxRowLength, violatedAmount: len(row), rowIndex: rowIndex}
		}
	}
	return nil
}

func gjsonResultsToColumns(gjsonResults []gjson.Result) [][]string {
	var results [][]string
	for _, gjsonResult := range gjsonResults {
		gjsonResult.ForEach(func(key, value gjson.Result) bool {
			row := getStringValuesFromNestedArrays(value, []string{})
			results = append(results, row)
			return true
		})
	}
	return results
}

// getStringValuesFromNestedArrays returns string values from a gjson.Result, doing this recursively.
// Within multipath expressions, it may happen that two-dimensional arrays are found. This function
// returns the string values of all nested arrays
func getStringValuesFromNestedArrays(value gjson.Result, values []string) []string {
	if value.String() == "" {
		return values
	}
	if !value.IsArray() {
		return append(values, value.String())
	}

	if value.IsArray() {
		value.ForEach(func(key, value gjson.Result) bool {
			values = getStringValuesFromNestedArrays(value, values)
			return true
		})
	}
	return values
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

// getStringsFromGJSONResult retrieves all results from a non-multipath
// expression and return a string array.
func getStringsFromGJSONResult(results []gjson.Result) []string {
	var stringResults []string
	for _, result := range results {
		result.ForEach(func(key, value gjson.Result) bool {
			stringResults = append(stringResults, value.String())
			return true
		})
	}
	return stringResults
}
