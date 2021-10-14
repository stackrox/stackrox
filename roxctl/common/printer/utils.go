package printer

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
)

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
	maxRowLength := 0
	for rowIndex, row := range rows {
		if rowIndex == 0 {
			maxRowLength = len(row)
		}
		if maxRowLength != len(row) {
			return errors.Errorf("number of values within each row are not matching. Expected "+
				"each row to hold %d values, but got %d values in a row",
				maxRowLength, len(row))
		}
	}
	return nil
}

func gjsonResultsToColumns(gjsonResults []gjson.Result) [][]string {
	var results [][]string
	for _, gjsonResult := range gjsonResults {
		gjsonResult.ForEach(func(key, value gjson.Result) bool {
			row := getColumnValues(value)
			results = append(results, row)
			return true
		})
	}
	return results
}

// getColumnValues will retrieve the column values of a gjson.Result and return a string array containing them.
func getColumnValues(value gjson.Result) []string {
	if !value.IsArray() {
		return nil
	}
	row := make([]string, 0, len(value.Array()))
	value.ForEach(func(key, value gjson.Result) bool {
		row = append(row, value.String())
		return true
	})
	return row
}

// getRowsFromColumns retrieves all rows from the given columns.
// NOTE: This function relies on the given columns array to not be jagged.
func getRowsFromColumns(columns [][]string) [][]string {
	var rows [][]string
	for colIndex := range columns[0] {
		row := make([]string, 0, len(columns[0]))
		for cellIndex := range columns {
			row = append(row, columns[cellIndex][colIndex])
		}
		rows = append(rows, row)
	}
	return rows
}
