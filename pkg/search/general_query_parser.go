package search

import (
	"errors"
	"sort"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/set"
)

// generalQueryParser provides parsing functionality for search requests.
type generalQueryParser struct {
	MatchAllIfEmpty     bool
	ExcludedFieldLabels set.StringSet
}

func getFieldMap(query string) map[FieldLabel][]string {
	pairs := splitQuery(query)

	fieldMap := make(map[FieldLabel][]string, len(pairs))
	for _, pair := range pairs {
		key, commaSeparatedValues, valid := parsePair(pair, false)
		if !valid || !IsValidFieldLabel(key) {
			continue
		}

		values := splitCommaSeparatedValues(commaSeparatedValues)
		fieldMap[FieldLabel(key)] = values
	}
	return fieldMap
}

// ParseFieldMap parses a query string into a map of field label to a list of field value strings
func ParseFieldMap(query string) (map[FieldLabel][]string, error) {
	fieldMap := getFieldMap(query)
	if len(fieldMap) == 0 {
		return nil, errox.InvalidArgs.CausedBy("after parsing, query is empty")
	}
	return fieldMap, nil
}

// SortFieldLabels takes a list of field labels and returns a sorted list of field labels
func SortFieldLabels(fieldLabels []FieldLabel) []FieldLabel {
	sort.Slice(fieldLabels, func(i, j int) bool {
		return fieldLabels[i] < fieldLabels[j]
	})
	return fieldLabels
}

// parse parses the input query.
func (pi generalQueryParser) parse(input string) (*v1.Query, error) {
	// Handle empty input query case.
	fieldMap := getFieldMap(input)
	if len(fieldMap) == 0 {
		if !pi.MatchAllIfEmpty {
			return nil, errors.New("parser not configured to handle empty queries")
		}
		return EmptyQuery(), nil
	}
	return pi.parseInternal(fieldMap)
}

func (pi generalQueryParser) parseInternal(fieldMap map[FieldLabel][]string) (*v1.Query, error) {
	qb := NewQueryBuilder()
	for fieldLabel, fieldValues := range fieldMap {
		if pi.ExcludedFieldLabels.Contains(fieldLabel.String()) {
			continue
		}
		qb.AddStrings(fieldLabel, fieldValues...)
	}
	return qb.ProtoQuery(), nil
}
