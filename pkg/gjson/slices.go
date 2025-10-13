package gjson

import (
	"encoding/json"

	"github.com/tidwall/gjson"
)

// SliceMapper is responsible for mapping a gjson.Result to slice representation in the form of multiple string
// slice results in a map
type SliceMapper struct {
	jsonPathExpressions map[string]string
	jsonBytes           []byte
}

// NewSliceMapper creates a SliceMapper which takes a json object and a map of GJSON compatible JSON expressions
func NewSliceMapper(jsonObj interface{}, jsonPathExpressionMap map[string]string) (*SliceMapper, error) {
	bytes, err := json.Marshal(jsonObj)
	if err != nil {
		return nil, err
	}

	return &SliceMapper{
		jsonBytes:           bytes,
		jsonPathExpressions: jsonPathExpressionMap,
	}, nil
}

// CreateSlices will execute all JSON path expressions in the map and return the results
// for each expression, retaining the key
func (s *SliceMapper) CreateSlices() map[string][]string {
	resultSlices := make(map[string][]string, len(s.jsonPathExpressions))

	for key, expression := range s.jsonPathExpressions {
		res := gjson.GetManyBytes(s.jsonBytes, expression)
		resultSlices[key] = getStringsFromGJSONResult(res)
	}

	return resultSlices
}

// getStringsFromGJSONResult retrieves all results from a non-multipath
// expression and return a string array.
func getStringsFromGJSONResult(results []gjson.Result) []string {
	var stringResults []string
	for _, result := range results {
		result.ForEach(func(key, value gjson.Result) bool {
			stringResults = append(stringResults, getStringValuesFromNestedArrays(value, []string{})...)
			return true
		})
	}
	// If we only have a single result _and_ that result is null, return a nil array.
	if len(results) == 1 && results[0].Type == gjson.Null {
		return nil
	}

	return stringResults
}
