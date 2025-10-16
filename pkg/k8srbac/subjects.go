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

// GetSubjectsAdjustedByKind returns subjects adjusted by kind scope.
// User and Group kind do not have namespace defined and such entities should not exist,
// but k8s storage allows it. Docs:
// https://kubernetes.io/docs/reference/kubernetes-api/authorization-resources/role-binding-v1/ -> subjects.namespace
func GetSubjectsAdjustedByKind(binding *storage.K8SRoleBinding) []*storage.Subject {
	if binding == nil {
		return nil
	}

	adjustedSubjectSet := NewSubjectSet()
	for _, subject := range binding.GetSubjects() {
		// Minimize number of CloneVT() calls.
		if subject.GetNamespace() != "" && (subject.GetKind() == storage.SubjectKind_USER || subject.GetKind() == storage.SubjectKind_GROUP) {
			adjustedSubject := subject.CloneVT()
			adjustedSubject.SetNamespace("")
			adjustedSubjectSet.Add(adjustedSubject)

			continue
		}

		adjustedSubjectSet.Add(subject)
	}

	return adjustedSubjectSet.ToSlice()
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

	subject := &storage.Subject{}
	subject.SetKind(storage.SubjectKind_SERVICE_ACCOUNT)
	subject.SetName(serviceAccount)
	subject.SetNamespace(deployment.GetNamespace())
	subject.SetClusterId(deployment.GetClusterId())
	subject.SetClusterName(deployment.GetClusterName())
	return subject
}

// GetSubjectForServiceAccount returns the subject represented by a service account.
func GetSubjectForServiceAccount(sa *storage.ServiceAccount) *storage.Subject {
	subject := &storage.Subject{}
	subject.SetKind(storage.SubjectKind_SERVICE_ACCOUNT)
	subject.SetName(sa.GetName())
	subject.SetNamespace(sa.GetNamespace())
	subject.SetClusterName(sa.GetClusterName())
	subject.SetClusterId(sa.GetClusterId())
	return subject
}

// GetAllSubjects get the subjects of the specified types in the referenced in a set of bindings.
func GetAllSubjects(bindings []*storage.K8SRoleBinding, kinds ...storage.SubjectKind) []*storage.Subject {
	subjectsSet := NewSubjectSet()
	for _, binding := range bindings {
		for _, subject := range GetSubjectsAdjustedByKind(binding) {
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
		for _, subject := range GetSubjectsAdjustedByKind(binding) {
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
