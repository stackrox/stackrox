package service

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/k8srbac"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func getSubject(subjectName string, roles []*storage.K8SRole, bindings []*storage.K8SRoleBinding) (*v1.GetSubjectResponse, error) {
	// Find the subject we want from the list of bindings.
	subjectToReturn, err := getSubjectToReturn(subjectName, bindings)
	if err != nil {
		return nil, err
	}

	// Separate bindings by cluster scoped and namespace scoped. Only use bindings with the role in it.
	clusterBindings := make([]*storage.K8SRoleBinding, 0)
	namespacedBindings := make(map[string][]*storage.K8SRoleBinding)
	for _, binding := range bindings {
		if !k8srbac.NewSubjectSet(binding.GetSubjects()...).Contains(subjectToReturn) {
			continue
		}
		if binding.GetClusterRole() {
			clusterBindings = append(clusterBindings, binding)
		} else {
			namespacedBindings[binding.GetNamespace()] = append(namespacedBindings[binding.GetNamespace()], binding)
		}
	}

	// transform the scoped bindings into cluster roles and roles per namespace.
	clusterRoles := k8srbac.NewEvaluator(roles, clusterBindings).RolesForSubject(subjectToReturn)
	namespacedRoles := make([]*v1.ScopedRoles, 0)
	for namespace, bindings := range namespacedBindings {
		namespacedRoles = append(namespacedRoles, &v1.ScopedRoles{
			Namespace: namespace,
			Roles:     k8srbac.NewEvaluator(roles, bindings).RolesForSubject(subjectToReturn),
		})
	}

	// Build response.
	return &v1.GetSubjectResponse{
		Subject:      subjectToReturn,
		ClusterRoles: clusterRoles,
		ScopedRoles:  namespacedRoles,
	}, nil
}

func getSubjectToReturn(subjectName string, bindings []*storage.K8SRoleBinding) (*storage.Subject, error) {
	// Find the subject we want.
	for _, binding := range bindings {
		for _, subject := range binding.GetSubjects() {
			// We only want to look for a user or a group.
			if subject.GetKind() != storage.SubjectKind_USER && subject.GetKind() != storage.SubjectKind_GROUP {
				continue
			}
			// Must have matching name (names are unique for groups and users).
			if subject.GetName() == subjectName {
				return subject, nil
			}
		}
	}
	return nil, status.Errorf(codes.NotFound, "subject not found: %s", subjectName)
}
