package blevesearch

import (
	"fmt"
	"strings"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
)

func evaluateEnum(value, field string, m map[int32]string) (query.Query, error) {
	value = strings.ToUpper(value)
	var matches []int32
	for val, t := range m {
		if strings.HasPrefix(t, value) {
			matches = append(matches, val)
		}
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("Couldn't find a value for '%s'", value)
	}
	dq := bleve.NewDisjunctionQuery()
	for _, s := range matches {
		dq.AddQuery(createNumericQuery(field, "=", floatPtr(float64(s))))
	}
	return dq, nil
}
