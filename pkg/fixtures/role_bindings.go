package fixtures

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/k8srbac"
	"github.com/stackrox/rox/pkg/uuid"
)

// GetScopedK8SRoleBinding returns a mock K8SRoleBinding belonging to the input scope.
func GetScopedK8SRoleBinding(id string, clusterID string, namespace string) *storage.K8SRoleBinding {
	return &storage.K8SRoleBinding{
		Id:        id,
		ClusterId: clusterID,
		Namespace: namespace,
	}
}

// GetK8sRoleBindingWithSubjects returns a K8sRoleBinding with given number of subjects
// SubjectKind will round-robin between service_account, user and group
func GetK8sRoleBindingWithSubjects(id, name, clusterID, clusterName, namespace string, clusterRole bool, numSubjects int) *storage.K8SRoleBinding {
	binding := &storage.K8SRoleBinding{
		Id:          id,
		Name:        name,
		ClusterName: clusterName,
		ClusterId:   clusterID,
		Namespace:   namespace,
		ClusterRole: clusterRole,
	}
	subjectKinds := []storage.SubjectKind{storage.SubjectKind_SERVICE_ACCOUNT, storage.SubjectKind_USER, storage.SubjectKind_GROUP}
	currKind := 0
	binding.Subjects = make([]*storage.Subject, 0, numSubjects)
	for i := 0; i < numSubjects; i++ {
		subjectName := fmt.Sprintf("%s_subject%d", name, i)
		subject := &storage.Subject{
			Id:          k8srbac.CreateSubjectID(clusterID, subjectName),
			Name:        subjectName,
			Kind:        subjectKinds[currKind],
			ClusterId:   clusterID,
			ClusterName: clusterName,
			Namespace:   namespace,
		}
		binding.Subjects = append(binding.Subjects, subject)
		currKind++
		if currKind >= len(subjectKinds) {
			currKind = 0
		}
	}
	return binding
}

// GetMultipleK8sRoleBindings returns given number of roleBindings, each with given number of subjects
// ClusterRole will toggle between true and false
func GetMultipleK8sRoleBindings(numBindings, numSubjectsPerBinding int) []*storage.K8SRoleBinding {
	clusterRole := true
	bindings := make([]*storage.K8SRoleBinding, 0, numBindings)
	for i := 0; i < numBindings; i++ {
		name := fmt.Sprintf("k8sRoleBinding%d", i)
		clusterName := fmt.Sprintf("cluster%d", i)
		namespace := fmt.Sprintf("namespace%d", i)
		bindings = append(bindings,
			GetK8sRoleBindingWithSubjects(
				uuid.NewV4().String(),
				name,
				uuid.NewV4().String(),
				clusterName,
				namespace,
				clusterRole,
				numSubjectsPerBinding))
		clusterRole = !clusterRole
	}
	return bindings
}
