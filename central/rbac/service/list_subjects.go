package service

import (
	"context"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	"github.com/stackrox/rox/central/rbac/service/mapping"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/k8srbac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/predicate"
)

var (
	subjectFactory = predicate.NewFactory("subject", (*storage.Subject)(nil))
)

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
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}
	return GetFilteredSubjects(parsed, subjectsToFilter)
}

type subjectSortAccessor func(s *storage.Subject) string

var subjectSortAccessors = map[string]subjectSortAccessor{
	strings.ToLower(search.SubjectKind.String()): func(s *storage.Subject) string { return s.GetKind().String() },
	strings.ToLower(search.SubjectName.String()): func(s *storage.Subject) string { return s.GetName() },
}

func sortSubjects(query *v1.Query, subjects []*storage.Subject) error {
	// Need to sort here based on the way that the subjects are derived
	if sortOptions := query.GetPagination().GetSortOptions(); len(sortOptions) > 0 {
		accessors := make([]subjectSortAccessor, 0, len(sortOptions))
		for _, s := range sortOptions {
			accessor, ok := subjectSortAccessors[strings.ToLower(s.Field)]
			if !ok {
				return errors.Errorf("sorting subjects by field %v is not supported", s.Field)
			}
			accessors = append(accessors, accessor)
		}
		sort.SliceStable(subjects, func(i, j int) bool {
			for idx, accessor := range accessors {
				val1, val2 := accessor(subjects[i]), accessor(subjects[j])
				if val1 != val2 {
					if sortOptions[idx].Reversed {
						return val1 > val2
					}
					return val1 < val2
				}
			}
			return false
		})
	}
	return nil
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
	if err := sortSubjects(query, filteredSubjects); err != nil {
		return nil, err
	}
	return filteredSubjects, nil
}

// SubjectSearcher encapsulates the derived subject searching from k8s role bindings
type SubjectSearcher struct {
	k8sRoleBindingDatastore datastore.DataStore
}

// NewSubjectSearcher takes in a k8s role binding and implements the derived subject searcher
func NewSubjectSearcher(k8sRoleBindingDatastore datastore.DataStore) *SubjectSearcher {
	return &SubjectSearcher{
		k8sRoleBindingDatastore: k8sRoleBindingDatastore,
	}
}

// Search implements the searcher interface
func (s *SubjectSearcher) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	subjectQuery, _ := search.FilterQueryWithMap(q, mapping.OptionsMap)
	pred, err := subjectFactory.GeneratePredicate(subjectQuery)
	if err != nil {
		return nil, err
	}

	bindings, err := s.k8sRoleBindingDatastore.SearchRawRoleBindings(ctx, q)
	if err != nil {
		return nil, err
	}
	// Subject return only users and groups, there is a separate resolver for service accounts.
	subjects := k8srbac.GetAllSubjects(bindings, storage.SubjectKind_USER, storage.SubjectKind_GROUP)
	// Sort first then evaluate to not run evaluation twice.
	// Sorting should be cheaper than reflect based evaluation
	if err := sortSubjects(q, subjects); err != nil {
		return nil, err
	}

	var results []search.Result
	for _, subject := range subjects {
		if result, match := pred.Evaluate(subject); match {
			results = append(results, *result)
		}
	}

	return results, nil
}

// Count returns the number of search results from the query
func (s *SubjectSearcher) Count(ctx context.Context, q *v1.Query) (int, error) {
	subjectQuery, _ := search.FilterQueryWithMap(q, mapping.OptionsMap)
	pred, err := subjectFactory.GeneratePredicate(subjectQuery)
	if err != nil {
		return 0, err
	}

	bindings, err := s.k8sRoleBindingDatastore.SearchRawRoleBindings(ctx, q)
	if err != nil {
		return 0, err
	}
	// Subject return only users and groups, there is a separate resolver for service accounts.
	subjects := k8srbac.GetAllSubjects(bindings, storage.SubjectKind_USER, storage.SubjectKind_GROUP)

	numResults := 0
	for _, subject := range subjects {
		if _, match := pred.Evaluate(subject); match {
			numResults++
		}
	}

	return numResults, nil
}

// SearchSubjects implements the search interface that returns v1.SearchResult
func (s *SubjectSearcher) SearchSubjects(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	subjectQuery, _ := search.FilterQueryWithMap(q, mapping.OptionsMap)
	pred, err := subjectFactory.GeneratePredicate(subjectQuery)
	if err != nil {
		return nil, err
	}

	bindings, err := s.k8sRoleBindingDatastore.SearchRawRoleBindings(ctx, q)
	if err != nil {
		return nil, err
	}
	// Subject return only users and groups, there is a separate resolver for service accounts.
	subjects := k8srbac.GetAllSubjects(bindings, storage.SubjectKind_USER, storage.SubjectKind_GROUP)
	// Sort first then evaluate to not run evaluation twice.
	// Sorting should be cheaper than reflect based evaluation
	if err := sortSubjects(q, subjects); err != nil {
		return nil, err
	}

	var searchResults []*v1.SearchResult
	for _, subject := range subjects {
		if pred.Matches(subject) {
			searchResults = append(searchResults, &v1.SearchResult{
				Id:       subject.Name,
				Name:     subject.Name,
				Category: v1.SearchCategory_SUBJECTS,
			})
		}
	}
	return searchResults, nil
}
