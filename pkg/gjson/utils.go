package gjson

import (
	"github.com/tidwall/gjson"
)

// getStringValuesFromNestedArrays returns string values from a gjson.Result, doing this recursively.
// Within multipath expressions, it may happen that two-dimensional arrays are found. This function
// returns the string values of all nested arrays
func getStringValuesFromNestedArrays(value gjson.Result, values []string) []string {
	// For tabular output, an empty string will be represented as
	// "-", if it would not, we would have a jagged array.
	if value.String() == "" {
		return append(values, "-")
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
