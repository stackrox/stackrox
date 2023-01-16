package service

import (
	"context"
	"math"

	k8sRoleDS "github.com/stackrox/rox/central/rbac/k8srole/datastore"
	k8sRoleBindingDS "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/k8srbac"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
)

var (
	log = logging.LoggerForModule()
)

func getSubjectFromStores(ctx context.Context, subjectName string, roleDS k8sRoleDS.DataStore, bindingDS k8sRoleBindingDS.DataStore) (*v1.GetSubjectResponse, error) {
	bindingsQuery := search.DisjunctionQuery(
		search.NewQueryBuilder().AddLinkedFields(
			[]search.FieldLabel{search.SubjectName, search.SubjectKind},
			[]string{search.ExactMatchString(subjectName), search.ExactMatchString(storage.SubjectKind_USER.String())}).ProtoQuery(),
		search.NewQueryBuilder().AddLinkedFields(
			[]search.FieldLabel{search.SubjectName, search.SubjectKind},
			[]string{search.ExactMatchString(subjectName), search.ExactMatchString(storage.SubjectKind_GROUP.String())}).ProtoQuery(),
	)
	bindingsQuery.Pagination = &v1.QueryPagination{
		Limit: math.MaxInt32,
	}
	relevantBindings, err := bindingDS.SearchRawRoleBindings(ctx, bindingsQuery)
	if err != nil || len(relevantBindings) == 0 {
		return nil, err
	}

	var subject *storage.Subject
	for _, subj := range relevantBindings[0].GetSubjects() {
		if subj.GetKind() != storage.SubjectKind_USER || subj.GetKind() != storage.SubjectKind_GROUP {
			continue
		}
		if subj.GetName() == subjectName {
			subject = subj
			break
		}
	}
	if subject == nil {
		log.Warnf("UNEXPECTED: subject %s not found in bindings even though there were %d relevant search results", subjectName, len(relevantBindings))
		return nil, nil
	}

	// Separate bindings by cluster scoped and namespace scoped. Only use bindings with the role in it.
	clusterBindings := make([]*storage.K8SRoleBinding, 0)
	namespacedBindings := make(map[string][]*storage.K8SRoleBinding)
	roleIDs := set.NewStringSet()
	for _, binding := range relevantBindings {
		roleIDs.Add(binding.GetRoleId())
		if k8srbac.IsClusterRoleBinding(binding) {
			clusterBindings = append(clusterBindings, binding)
		} else {
			namespacedBindings[binding.GetNamespace()] = append(namespacedBindings[binding.GetNamespace()], binding)
		}
	}

	rolesQuery := search.NewQueryBuilder().AddExactMatches(search.RoleID, roleIDs.AsSlice()...).ProtoQuery()
	rolesQuery.Pagination = &v1.QueryPagination{
		Limit: math.MaxInt32,
	}
	relevantRoles, err := roleDS.SearchRawRoles(ctx, rolesQuery)
	if err != nil {
		return nil, err
	}

	// transform the scoped bindings into cluster roles and roles per namespace.
	clusterRoles := k8srbac.NewEvaluator(relevantRoles, clusterBindings).RolesForSubject(subject)
	namespacedRoles := make([]*v1.ScopedRoles, 0)
	for namespace, bindings := range namespacedBindings {
		namespacedRoles = append(namespacedRoles, &v1.ScopedRoles{
			Namespace: namespace,
			Roles:     k8srbac.NewEvaluator(relevantRoles, bindings).RolesForSubject(subject),
		})
	}

	// Build response.
	return &v1.GetSubjectResponse{
		Subject:      subject,
		ClusterRoles: clusterRoles,
		ScopedRoles:  namespacedRoles,
	}, nil
}
