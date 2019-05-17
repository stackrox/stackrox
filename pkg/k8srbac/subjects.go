package k8srbac

import (
	"github.com/stackrox/rox/generated/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetSubjectForDeployment returns the subject represented by a deployment.
func GetSubjectForDeployment(deployment *storage.Deployment) *storage.Subject {
	var serviceAccount string
	if deployment.GetServiceAccount() == "" {
		serviceAccount = "default"
	} else {
		serviceAccount = deployment.GetServiceAccount()
	}

	return &storage.Subject{
		Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
		Name:      serviceAccount,
		Namespace: deployment.GetNamespace(),
	}
}

// GetSubjectForServiceAccount returns the subject represented by a service account.
func GetSubjectForServiceAccount(sa *storage.ServiceAccount) *storage.Subject {
	return &storage.Subject{
		Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
		Name:      sa.GetName(),
		Namespace: sa.GetNamespace(),
	}
}

// GetAllSubjects get the subjects of the specified types in the referenced in a set of bindings.
func GetAllSubjects(bindings []*storage.K8SRoleBinding, kinds ...storage.SubjectKind) []*storage.Subject {
	subjectsSet := NewSubjectSet()
	for _, binding := range bindings {
		for _, subject := range binding.GetSubjects() {
			for _, kind := range kinds {
				if subject.GetKind() == kind {
					subjectsSet.Add(subject)
					break
				}
			}
		}
	}
	return subjectsSet.ToSlice()
}

// GetSubject gets the subject of the specified name if referenced in a set of bindings.
func GetSubject(subjectName string, bindings []*storage.K8SRoleBinding) (*storage.Subject, bool, error) {
	// Find the subject we want.
	for _, binding := range bindings {
		for _, subject := range binding.GetSubjects() {
			// We only want to look for a user or a group.
			if subject.GetKind() != storage.SubjectKind_USER && subject.GetKind() != storage.SubjectKind_GROUP {
				continue
			}
			// Must have matching name (names are unique for groups and users).
			if subject.GetName() == subjectName {
				return subject, true, nil
			}
		}
	}
	return nil, false, status.Errorf(codes.NotFound, "subject not found: %s", subjectName)
}
