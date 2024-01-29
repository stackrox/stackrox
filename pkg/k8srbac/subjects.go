package k8srbac

import (
	"encoding/base64"
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/stringutils"
)

// CreateSubjectID creates a composite ID from cluster id and subject
func CreateSubjectID(clusterID, subjectName string) string {
	clusterEncoded := base64.URLEncoding.EncodeToString([]byte(clusterID))
	subjectEncoded := base64.URLEncoding.EncodeToString([]byte(subjectName))
	return fmt.Sprintf("%s:%s", clusterEncoded, subjectEncoded)
}

// SplitSubjectID returns the components of the ID
func SplitSubjectID(id string) (string, string, error) {
	clusterEncoded, subjectEncoded := stringutils.Split2(id, ":")
	clusterID, err := base64.URLEncoding.DecodeString(clusterEncoded)
	if err != nil {
		return "", "", err
	}
	subjectName, err := base64.URLEncoding.DecodeString(subjectEncoded)
	if err != nil {
		return "", "", err
	}
	return string(clusterID), string(subjectName), nil
}

// GetSubjectForDeployment returns the subject represented by a deployment.
func GetSubjectForDeployment(deployment *storage.Deployment) *storage.Subject {
	var serviceAccount string
	if deployment.GetServiceAccount() == "" {
		serviceAccount = "default"
	} else {
		serviceAccount = deployment.GetServiceAccount()
	}

	return &storage.Subject{
		Kind:        storage.SubjectKind_SERVICE_ACCOUNT,
		Name:        serviceAccount,
		Namespace:   deployment.GetNamespace(),
		ClusterId:   deployment.GetClusterId(),
		ClusterName: deployment.GetClusterName(),
	}
}

// GetSubjectForServiceAccount returns the subject represented by a service account.
func GetSubjectForServiceAccount(sa *storage.ServiceAccount) *storage.Subject {
	return &storage.Subject{
		Kind:        storage.SubjectKind_SERVICE_ACCOUNT,
		Name:        sa.GetName(),
		Namespace:   sa.GetNamespace(),
		ClusterName: sa.GetClusterName(),
		ClusterId:   sa.GetClusterId(),
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
	return nil, false, errors.Wrapf(errox.NotFound, "subject not found: %s", subjectName)
}
