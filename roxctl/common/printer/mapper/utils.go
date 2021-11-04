package mapper

import (
	"github.com/tidwall/gjson"
)

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
