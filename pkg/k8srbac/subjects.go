package k8srbac

import (
	"github.com/stackrox/rox/generated/storage"
)

// GetSubjectForDeployment returns the subject represented by a deployment.
func GetSubjectForDeployment(deployment *storage.Deployment) *storage.Subject {
	if deployment.GetServiceAccount() == "" {
		return nil
	}
	return &storage.Subject{
		Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
		Name:      deployment.GetServiceAccount(),
		Namespace: deployment.GetNamespace(),
	}
}
