package k8srbac

import (
	"github.com/stackrox/rox/generated/storage"
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
