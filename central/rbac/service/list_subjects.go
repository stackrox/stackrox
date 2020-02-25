package service

import (
	"github.com/stackrox/rox/central/rbac/service/mapping"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/k8srbac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/predicate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var subjectFactory = predicate.NewFactory("subject", (*storage.Subject)(nil))

func listSubjects(rawQuery *v1.RawQuery, roles []*storage.K8SRole, bindings []*storage.K8SRoleBinding) (*v1.ListSubjectsResponse, error) {
	subjectsToList, err := getFilteredSubjectsByRoleBinding(rawQuery, bindings)
	if err != nil {
		return nil, err
	}

	// Collect all of the subjects with at least one role in the set of roles and bindings.
	evaluator := k8srbac.NewEvaluator(roles, bindings)
	subjectsAndRoles := make([]*v1.SubjectAndRoles, 0, len(subjectsToList))
	for _, subject := range subjectsToList {
		roles := evaluator.RolesForSubject(subject)
		subjectsAndRoles = append(subjectsAndRoles, &v1.SubjectAndRoles{
			Subject: subject,
			Roles:   roles,
		})
	}

	// Build response.
	return &v1.ListSubjectsResponse{
		SubjectAndRoles: subjectsAndRoles,
	}, nil
}

// Filter subjects referenced in a set of bindings with a raw search query.
func getFilteredSubjectsByRoleBinding(rawQuery *v1.RawQuery, bindings []*storage.K8SRoleBinding) ([]*storage.Subject, error) {
	subjectsToFilter := k8srbac.GetAllSubjects(bindings, storage.SubjectKind_USER, storage.SubjectKind_GROUP)
	if len(subjectsToFilter) == 0 {
		return nil, nil
	}

	// Filter the input query to only have subject fields.
	subjectQuery := &v1.RawQuery{
		Query: search.FilterFields(rawQuery.GetQuery(), func(field string) bool {
			_, isSubjectField := mapping.OptionsMap.Get(field)
			return isSubjectField
		}),
	}
	if subjectQuery.GetQuery() == "" {
		return subjectsToFilter, nil
	}

	// Parse the query we will filter with.
	var parsed *v1.Query
	parsed, err := search.ParseQuery(subjectQuery.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return GetFilteredSubjects(parsed, subjectsToFilter)
}

// GetFilteredSubjects filters subjects based on a proto query. This function modifies subjectsToFilter
func GetFilteredSubjects(query *v1.Query, subjectsToFilter []*storage.Subject) ([]*storage.Subject, error) {
	pred, err := subjectFactory.GeneratePredicate(query)
	if err != nil {
		return nil, err
	}

	filteredSubjects := subjectsToFilter[:0]
	for _, subject := range subjectsToFilter {
		if pred.Matches(subject) {
			filteredSubjects = append(filteredSubjects, subject)
		}
	}
	return filteredSubjects, nil
}
