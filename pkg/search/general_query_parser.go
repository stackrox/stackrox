package search

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	v1 "github.com/stackrox/rox/generated/api/v1"
)

// generalQueryParser provides parsing functionality for search requests.
type generalQueryParser struct {
	HighlightFields bool
	LinkFields      bool
	MatchAllIfEmpty bool
}

// ParseFieldMap parses a query string into a map of field label to a list of field value strings
func ParseFieldMap(query string) (map[FieldLabel][]string, error) {
	pairs := strings.Split(query, "+")

	var anyValid bool
	fieldMap := make(map[FieldLabel][]string, len(pairs))
	for _, pair := range pairs {
		key, commaSeparatedValues, valid := parsePair(pair, false)
		if !valid {
			continue
		}
		values := strings.Split(commaSeparatedValues, ",")
		fieldMap[FieldLabel(key)] = values
		anyValid = true
	}
	if !anyValid {
		return nil, errors.New("after parsing, query is empty")
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

// Parse parses the input query.
func (pi generalQueryParser) parse(input string) (*v1.Query, error) {
	// Handle empty input query case.
	if len(input) == 0 && !pi.MatchAllIfEmpty {
		return nil, errors.New("parser not configured to handle empty queries")
	} else if len(input) == 0 {
		return EmptyQuery(), nil
	}
	// Have a filled query, parse it.
	if pi.LinkFields {
		return pi.parseInternalLinked(input)
	}
	return pi.parseInternal(input)
}

func (pi generalQueryParser) parseInternal(query string) (*v1.Query, error) {
	fieldMap, err := ParseFieldMap(query)
	if err != nil {
		return nil, err
	}
	qb := NewQueryBuilder()
	for fieldLabel, fieldValues := range fieldMap {
		qb.AddStrings(fieldLabel, fieldValues...)
		if pi.HighlightFields {
			qb.MarkHighlighted(fieldLabel)
		}
	}
	return qb.ProtoQuery(), nil
}

func (pi generalQueryParser) parseInternalLinked(query string) (*v1.Query, error) {
	pairs := strings.Split(query, "+")

	var anyValid bool
	var fields []FieldLabel
	var values []string
	for _, pair := range pairs {
		key, commaSeparatedValues, valid := parsePair(pair, false)
		if !valid {
			continue
		}
		fieldValues := strings.Split(commaSeparatedValues, ",")
		if len(fieldValues) != 1 {
			return nil, fmt.Errorf("field %s has multiple values for a linked field which is not supported", key)
		}
		fields = append(fields, FieldLabel(key))
		values = append(values, fieldValues[0])
		anyValid = true
	}
	if !anyValid {
		return nil, errors.New("after parsing, query is empty")
	}

	qb := NewQueryBuilder()
	if pi.HighlightFields {
		qb.AddLinkedFieldsHighlighted(fields, values)
	} else {
		qb.AddLinkedFields(fields, values)
	}
	return qb.ProtoQuery(), nil
}
