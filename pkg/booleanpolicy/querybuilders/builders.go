package querybuilders

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/booleanpolicy/query"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/search/predicate/basematchers"
	"github.com/stackrox/stackrox/pkg/stringutils"
)

var (
	operatorProtoMap = map[storage.BooleanOperator]query.Operator{
		storage.BooleanOperator_OR:  query.Or,
		storage.BooleanOperator_AND: query.And,
	}
)

func valueToStringExact(value string) string {
	return search.ExactMatchString(value)
}

func valueToStringRegex(value string) string {
	if strings.HasPrefix(value, search.RegexPrefix) {
		return value
	}
	return search.RegexPrefix + value
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
