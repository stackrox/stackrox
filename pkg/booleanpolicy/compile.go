package booleanpolicy

import (
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/booleanpolicy/query"
	"github.com/stackrox/stackrox/pkg/set"
)

func sectionToQuery(section *storage.PolicySection, stage storage.LifecycleStage) (*query.Query, error) {
	if len(section.GetPolicyGroups()) == 0 {
		return nil, errors.New("no groups")
	}
	fieldQueries, err := sectionToFieldQueries(section)
	if err != nil {
		return nil, err
	}
	contextQueries := constructRemainingContextQueries(stage, section, fieldQueries)
	fieldQueries = append(fieldQueries, contextQueries...)

	return &query.Query{FieldQueries: fieldQueries}, nil
}

func sectionTypeToFieldQueries(section *storage.PolicySection, fieldType RuntimeFieldType) ([]*query.FieldQuery, error) {
	fieldQueries := make([]*query.FieldQuery, 0, len(section.GetPolicyGroups()))
	for _, group := range section.GetPolicyGroups() {
		if !FieldMetadataSingleton().FieldIsOfType(group.GetFieldName(), fieldType) {
			continue
		}
		fqs, err := policyGroupToFieldQueries(group)
		if err != nil {
			return nil, errors.Wrapf(err, "constructing query for group %s", group.GetFieldName())
		}
		fieldQueries = append(fieldQueries, fqs...)
	}
	return fieldQueries, nil
}

func sectionToFieldQueries(section *storage.PolicySection) ([]*query.FieldQuery, error) {
	fieldQueries := make([]*query.FieldQuery, 0, len(section.GetPolicyGroups()))
	for _, group := range section.GetPolicyGroups() {
		fqs, err := policyGroupToFieldQueries(group)
		if err != nil {
			return nil, errors.Wrapf(err, "constructing query for group %s", group.GetFieldName())
		}
		fieldQueries = append(fieldQueries, fqs...)
	}
	return fieldQueries, nil
}

func policyGroupToFieldQueries(group *storage.PolicyGroup) ([]*query.FieldQuery, error) {
	if len(group.GetValues()) == 0 {
		return nil, errors.New("no values")
	}

	metadata, err := FieldMetadataSingleton().findField(group.GetFieldName())
	if err != nil {
		return nil, errors.Errorf("no QB known for group %q", group.GetFieldName())
	}

	if metadata == nil || metadata.qb == nil {
		return nil, errors.Errorf("no QB known for group %q", group.GetFieldName())
	}

	if metadata.negationForbidden && group.GetNegate() {
		return nil, errors.Errorf("invalid group: negation not allowed for field %s", group.GetFieldName())
	}
	if metadata.operatorsForbidden && len(group.GetValues()) != 1 {
		return nil, errors.Errorf("invalid group: operators not allowed for field %s", group.GetFieldName())
	}

	fqs := metadata.qb.FieldQueriesForGroup(group)
	if len(fqs) == 0 {
		return nil, errors.New("invalid group: no queries formed")
	}

	return fqs, nil
}

func matchAllQueryForField(fieldName string) *query.FieldQuery {
	return &query.FieldQuery{
		Field:    fieldName,
		MatchAll: true,
	}
}

// Add matchAll field queries for context fields that are not already included fieldQueries
func constructRemainingContextQueries(stage storage.LifecycleStage, section *storage.PolicySection, fieldQueries []*query.FieldQuery) []*query.FieldQuery {
	fieldSet := set.NewStringSet()
	for _, query := range fieldQueries {
		fieldSet.Add(query.Field)
	}
	contextFieldSet := set.NewStringSet()
	for _, group := range section.GetPolicyGroups() {
		if metadata, err := FieldMetadataSingleton().findField(group.GetFieldName()); err == nil {
			if contextFieldsToAdd, ok := metadata.contextFields[stage]; ok {
				for _, contextField := range contextFieldsToAdd.AsSlice() {
					contextFieldSet.Add(contextField)
				}
			}
		}
	}
	var contextQueries []*query.FieldQuery
	for contextField := range contextFieldSet {
		if !fieldSet.Contains(contextField) {
			contextQueries = append(contextQueries, matchAllQueryForField(contextField))
		}
	}
	return contextQueries
}
