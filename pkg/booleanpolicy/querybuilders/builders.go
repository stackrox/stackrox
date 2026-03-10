package querybuilders

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator/mapeval"
	"github.com/stackrox/rox/pkg/booleanpolicy/query"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/predicate/basematchers"
	"github.com/stackrox/rox/pkg/stringutils"
)

var (
	operatorProtoMap = map[storage.BooleanOperator]query.Operator{
		storage.BooleanOperator_OR:  query.Or,
		storage.BooleanOperator_AND: query.And,
	}
)

func valueToPathGlob(value string) string {
	if strings.HasPrefix(value, search.GlobPrefix) {
		return value
	}
	return search.GlobPrefix + value
}

func valueToStringExact(value string) string {
	return search.ExactMatchString(value)
}

func valueToStringRegex(value string) string {
	if strings.HasPrefix(value, search.RegexPrefix) {
		return value
	}
	return search.RegexPrefix + value
}

func valueToStringContainsRegex(value string) string {
	if strings.HasPrefix(value, search.RegexPrefix) {
		return strings.Join([]string{search.RegexPrefix, search.ContainsPrefix, strings.TrimPrefix(value, search.RegexPrefix)}, "")
	}
	return strings.Join([]string{search.RegexPrefix, search.ContainsPrefix, value}, "")
}

func negateBool(value string) (string, error) {
	b, err := strconv.ParseBool(value)
	if err != nil {
		return "", err
	}

	return strconv.FormatBool(!b), nil
}

func mapValues(group *storage.PolicyGroup, f func(string) string) []string {
	out := make([]string, 0, len(group.GetValues()))
	for _, v := range group.GetValues() {
		var mappedValue string
		if f != nil {
			mappedValue = f(v.GetValue())
		} else {
			mappedValue = v.GetValue()
		}
		out = append(out, mappedValue)
	}
	return out
}

// A QueryBuilder builds queries for a specific policy group.
type QueryBuilder interface {
	FieldQueriesForGroup(group *storage.PolicyGroup) []*query.FieldQuery
}

type queryBuilderFunc func(group *storage.PolicyGroup) []*query.FieldQuery

func (f queryBuilderFunc) FieldQueriesForGroup(group *storage.PolicyGroup) []*query.FieldQuery {
	return f(group)
}

type fieldLabelQueryBuilder struct {
	fieldLabel   search.FieldLabel
	valueMapFunc func(string) string
}

func (f *fieldLabelQueryBuilder) FieldQueriesForGroup(group *storage.PolicyGroup) []*query.FieldQuery {
	return []*query.FieldQuery{fieldQueryFromGroup(group, f.fieldLabel, f.valueMapFunc)}
}

func ForFieldLabelFilePath(label search.FieldLabel) QueryBuilder {
	return &fieldLabelQueryBuilder{fieldLabel: label, valueMapFunc: valueToPathGlob}
}

// ForFieldLabelExact returns a query builder that simply queries for the exact field value with the given search field label.
func ForFieldLabelExact(label search.FieldLabel) QueryBuilder {
	return &fieldLabelQueryBuilder{fieldLabel: label, valueMapFunc: valueToStringExact}
}

// ForFieldLabel returns a query builder that does a prefix match for the field value with the given search field label.
func ForFieldLabel(label search.FieldLabel) QueryBuilder {
	return &fieldLabelQueryBuilder{fieldLabel: label}
}

// ForFieldLabelRegex is like ForFieldLabel, but does a regex match.
func ForFieldLabelRegex(label search.FieldLabel) QueryBuilder {
	return &fieldLabelQueryBuilder{fieldLabel: label, valueMapFunc: valueToStringRegex}
}

// ForFieldLabelContainsRegex is like ForFieldLabel, but does a contains regex match.
func ForFieldLabelContainsRegex(label search.FieldLabel) QueryBuilder {
	return &fieldLabelQueryBuilder{fieldLabel: label, valueMapFunc: valueToStringContainsRegex}
}

// ForFieldLabelUpper is like ForFieldLabel, but does a match after converting the query to upper-case.
func ForFieldLabelUpper(label search.FieldLabel) QueryBuilder {
	return &fieldLabelQueryBuilder{fieldLabel: label, valueMapFunc: strings.ToUpper}
}

// ForFieldLabelMap is like ForFieldLabel, but does a map match where the query is converted appropriately.
func ForFieldLabelMap(label search.FieldLabel, qb func(string, string) string) QueryBuilder {
	mapFunc := func(value string) string {
		key, val := stringutils.Split2(value, "=")
		return qb(key, val)
	}

	return &fieldLabelQueryBuilder{fieldLabel: label, valueMapFunc: mapFunc}
}

// ForFieldLabelMapRequired builds a query for "required" map fields (Required Label,
// Required Annotation, Required Image Label). When values are OR'd, they are combined
// into a single conjunction group so the map evaluator checks all constraints together:
// a violation fires only when NONE of the required entries are present. Without this,
// each value would become an independent MapShouldNotContain matcher, and OR at the
// evaluator level would invert the semantics via De Morgan's law (violation if ANY is
// missing instead of violation if ALL are missing).
func ForFieldLabelMapRequired(label search.FieldLabel) QueryBuilder {
	return queryBuilderFunc(func(group *storage.PolicyGroup) []*query.FieldQuery {
		if group.GetBooleanOperator() == storage.BooleanOperator_OR && len(group.GetValues()) > 1 {
			parts := make([]string, 0, len(group.GetValues()))
			for _, v := range group.GetValues() {
				key, val := stringutils.Split2(v.GetValue(), "=")
				parts = append(parts, query.MapShouldNotContain(key, val))
			}
			combined := strings.Join(parts, mapeval.ConjunctionMarker)
			return []*query.FieldQuery{{
				Field:    label.String(),
				Values:   []string{combined},
				Operator: operatorProtoMap[group.GetBooleanOperator()],
				Negate:   group.GetNegate(),
			}}
		}
		mapFunc := func(value string) string {
			key, val := stringutils.Split2(value, "=")
			return query.MapShouldNotContain(key, val)
		}
		return []*query.FieldQuery{fieldQueryFromGroup(group, label, mapFunc)}
	})
}

// ForDays is like ForFieldLabel, but does a match depending on current_time - query(in days) >= event_time.
func ForDays(label search.FieldLabel) QueryBuilder {
	appendOpFunc := func(value string) string {
		return fmt.Sprintf("%s %s", basematchers.GreaterThanOrEqualTo, value)
	}

	return &fieldLabelQueryBuilder{fieldLabel: label, valueMapFunc: appendOpFunc}
}

// ForFieldLabelBoolean is like ForFieldLabel, but does a match after appropriately converting the boolean value.
func ForFieldLabelBoolean(label search.FieldLabel, negateValue bool) QueryBuilder {
	return queryBuilderFunc(func(group *storage.PolicyGroup) []*query.FieldQuery {
		mappedValues := make([]string, 0, len(group.GetValues()))
		for _, value := range group.GetValues() {
			valueStr := value.GetValue()
			if negateValue {
				var err error
				valueStr, err = negateBool(value.GetValue())
				if err != nil {
					return nil
				}
			}

			mappedValues = append(mappedValues, valueStr)
		}

		return []*query.FieldQuery{{
			Field:  label.String(),
			Values: mappedValues,
			Negate: group.GetNegate(),
		}}
	})
}

// ForFieldLabelNil checks whether a particular field label is set to nil.
func ForFieldLabelNil(label search.FieldLabel) QueryBuilder {
	return queryBuilderFunc(func(group *storage.PolicyGroup) []*query.FieldQuery {
		values := group.GetValues()
		if len(values) != 1 {
			return nil
		}

		mappedValues := make([]string, 0, len(group.GetValues()))
		negate := false
		for _, value := range values {
			valueStr := value.GetValue()
			b, err := strconv.ParseBool(valueStr)
			if err != nil {
				return nil
			}

			if !b {
				negate = true
			}

			mappedValues = append(mappedValues, search.NullString)
		}

		return []*query.FieldQuery{{
			Field:  label.String(),
			Values: mappedValues,
			Negate: negate != group.GetNegate(),
		}}
	})
}

func fieldQueryFromGroup(group *storage.PolicyGroup, label search.FieldLabel, mapFunc func(string) string) *query.FieldQuery {
	return &query.FieldQuery{
		Field:    label.String(),
		Values:   mapValues(group, mapFunc),
		Operator: operatorProtoMap[group.GetBooleanOperator()],
		Negate:   group.GetNegate(),
	}
}
