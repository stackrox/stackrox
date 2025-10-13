package gjson

import (
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
)

// emptyReplacement is used as a string representation for an empty gjson.Result.
const emptyReplacement = "-"

// getStringValuesFromNestedArrays returns string values from a gjson.Result, doing this recursively.
// Within multipath expressions, it may happen that two-dimensional arrays are found. This function
// returns the string values of all nested arrays.
func getStringValuesFromNestedArrays(value gjson.Result, values []string) []string {
	// For tabular output, an empty string will be represented as
	// "-", if it would not, we would have a jagged array.
	if value.String() == "" {
		return append(values, emptyReplacement)
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

// Expression is a gjson expression that can be used in multipath expressions.
// The key is optional, in case a different, custom key should be used in the resulting JSON.
type Expression struct {
	Key        string
	Expression string
}

// MultiPathExpression will generate a gjson compatible multipath expression string.
func MultiPathExpression(modifier string, expressions ...Expression) string {
	var sb strings.Builder
	sb.WriteString("{")
	for i, expression := range expressions {
		if i != 0 {
			sb.WriteString(",")
		}
		if expression.Key != "" {
			sb.WriteString(fmt.Sprintf("%q:", expression.Key))
		}
		sb.WriteString(expression.Expression)
	}
	sb.WriteString("}")
	if modifier != "" {
		sb.WriteString(fmt.Sprintf(".%s", modifier))
	}
	return sb.String()
}
