package service

import (
	"time"

	"github.com/blevesearch/bleve"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/k8srbac"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func listSubjects(rawQuery *v1.RawQuery, roles []*storage.K8SRole, bindings []*storage.K8SRoleBinding) (*v1.ListSubjectsResponse, error) {
	subjectsToList, err := getFilteredSubjects(rawQuery, bindings)
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
func getFilteredSubjects(rawQuery *v1.RawQuery, bindings []*storage.K8SRoleBinding) ([]*storage.Subject, error) {
	subjectsToFilter := k8srbac.GetAllSubjects(bindings, storage.SubjectKind_USER, storage.SubjectKind_GROUP)
	if len(subjectsToFilter) == 0 {
		return nil, nil
	}

	// Filter the input query to only have subject fields.
	subjectQuery := &v1.RawQuery{
		Query: search.FilterFields(rawQuery.GetQuery(), func(field string) bool {
			_, isSubjectField := optionsMap.Get(field)
			return isSubjectField
		}),
	}
	if subjectQuery.GetQuery() == "" {
		return subjectsToFilter, nil
	}

	// Parse the query we will filter with.
	var parsed *v1.Query
	parsed, err := search.ParseRawQueryOrEmpty(subjectQuery.GetQuery())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Create a temporary index.
	tempIndex, err := globalindex.MemOnlyIndex()
	defer utils.IgnoreError(tempIndex.Close)

	if err != nil {
		return nil, errors.Wrapf(err, "initializing temp index")
	}
	defer utils.IgnoreError(tempIndex.Close)
	tempIndexer := indexerImpl{index: tempIndex}

	// Index all of the subjects, and map by name.
	subjectsByName := make(map[string]*storage.Subject)
	for _, subject := range subjectsToFilter {
		subjectsByName[subject.GetName()] = subject
		if err := tempIndexer.Add(subject); err != nil {
			return nil, errors.Wrapf(err, "inserting into temp index")
		}
	}

	// Run the search.
	resultNames, err := tempIndexer.Search(parsed)
	if err != nil {
		return nil, errors.Wrapf(err, "searching temp index")
	}

	// Collect the resulting subjects of the search.
	subjectsToUse := make([]*storage.Subject, 0)
	for _, result := range resultNames {
		subjectsToUse = append(subjectsToUse, subjectsByName[result.ID])
	}
	return subjectsToUse, nil
}

// Utils to temporarily index subjects.
///////////////////////////////////////

var optionsMap = blevesearch.Walk(v1.SearchCategory_SUBJECTS, "subject", (*storage.Subject)(nil))

// Wrapper to index subjects within.
type subjectWrapper struct {
	// Json name of this field must match what is used in k8srole/search/options/map
	*storage.Subject `json:"subject"`
	Type             string `json:"type"`
}

func wrap(subject *storage.Subject) *subjectWrapper {
	return &subjectWrapper{Type: v1.SearchCategory_SUBJECTS.String(), Subject: subject}
}

// Index implementation
type indexerImpl struct {
	index bleve.Index
}

func (i *indexerImpl) Add(subject *storage.Subject) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Add, "Subject")
	return i.index.Index(subject.GetName(), wrap(subject))
}

func (i *indexerImpl) Search(subjectQuery *v1.Query) ([]searchPkg.Result, error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Search, "Subject")
	return blevesearch.RunSearchRequest(v1.SearchCategory_SUBJECTS, subjectQuery, i.index, optionsMap)
}
