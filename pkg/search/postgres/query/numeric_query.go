package pgsearch

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/search"
)

type prefixAndInversion struct {
	prefix    string
	inversion string
}

var (
	prefixesAndInversions = []prefixAndInversion{
		{"<", ">="},
		{">", "<="},
	}

	validPrefixesSortedByLengthDec = func() []string {
		var validPrefixes []string
		for _, pAndI := range prefixesAndInversions {
			validPrefixes = append(validPrefixes, pAndI.prefix)
			validPrefixes = append(validPrefixes, pAndI.inversion)
		}
		validPrefixes = append(validPrefixes, "==")
		sort.Slice(validPrefixes, func(i, j int) bool {
			return len(validPrefixes[i]) > len(validPrefixes[j])
		})
		return validPrefixes
	}()
)

func parseNumericPrefix(value string) (prefix string, trimmedValue string) {
	for _, prefix := range validPrefixesSortedByLengthDec {
		if strings.HasPrefix(value, prefix) {
			return prefix, strings.TrimPrefix(value, prefix)
		}
	}
	return "", value
}

func parseNumericStringToPtr(s string) (float64, error) {
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	return val, nil
}

func createNumericQuery(root string, _ *search.Field, prefix string, value float64) *QueryEntry {
	var valueStr string
	if _, fraction := math.Modf(value); fraction > 0 {
		valueStr = fmt.Sprintf("%0.2f", value)
	} else {
		valueStr = fmt.Sprintf("%0.0f", value)
	}

	if prefix == "" {
		prefix = "="
	}
	return &QueryEntry{
		Query:  fmt.Sprintf("%s %s $$", root, prefix),
		Values: []interface{}{valueStr},
	}
}

func newNumericQuery(table string, field *search.Field, value string, modifiers ...search.QueryModifier) (*QueryEntry, error) {
	if len(modifiers) > 0 {
		return nil, errors.Errorf("modifiers not supported for numeric query: %+v", modifiers)
	}
	prefix, trimmedValue := parseNumericPrefix(value)
	valuePtr, err := parseNumericStringToPtr(trimmedValue)
	if err != nil {
		return nil, err
	}
	return createNumericQuery(table, field, prefix, valuePtr), nil
}
