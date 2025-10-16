package fixtures

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/k8srbac"
	"github.com/stackrox/rox/pkg/uuid"
)

// GetScopedK8SRoleBinding returns a mock K8SRoleBinding belonging to the input scope.
func GetScopedK8SRoleBinding(id string, clusterID string, namespace string) *storage.K8SRoleBinding {
	k8srb := &storage.K8SRoleBinding{}
	k8srb.SetId(id)
	k8srb.SetClusterId(clusterID)
	k8srb.SetNamespace(namespace)
	return k8srb
}

// GetK8sRoleBindingWithSubjects returns a K8sRoleBinding with given number of subjects
// SubjectKind will round-robin between service_account, user and group
func GetK8sRoleBindingWithSubjects(id, name, clusterID, clusterName, namespace string, clusterRole bool, numSubjects int) *storage.K8SRoleBinding {
	binding := &storage.K8SRoleBinding{}
	binding.SetId(id)
	binding.SetName(name)
	binding.SetClusterName(clusterName)
	binding.SetClusterId(clusterID)
	binding.SetNamespace(namespace)
	binding.SetClusterRole(clusterRole)
	subjectKinds := []storage.SubjectKind{storage.SubjectKind_SERVICE_ACCOUNT, storage.SubjectKind_USER, storage.SubjectKind_GROUP}
	currKind := 0
	binding.SetSubjects(make([]*storage.Subject, 0, numSubjects))
	for i := 0; i < numSubjects; i++ {
		subjectName := fmt.Sprintf("%s_subject%d", name, i)
		subject := &storage.Subject{}
		subject.SetId(k8srbac.CreateSubjectID(clusterID, subjectName))
		subject.SetName(subjectName)
		subject.SetKind(subjectKinds[currKind])
		subject.SetClusterId(clusterID)
		subject.SetClusterName(clusterName)
		subject.SetNamespace(namespace)
		binding.SetSubjects(append(binding.GetSubjects(), subject))
		currKind++
		if currKind >= len(subjectKinds) {
			currKind = 0
		}
	}
	return binding
}

// GetMultipleK8sRoleBindings returns given number of roleBindings, each with given number of subjects
// The cluster role property will toggle from false to true.
func GetMultipleK8sRoleBindings(numBindings, numSubjectsPerBinding int) []*storage.K8SRoleBinding {
	return GetMultipleK8sRoleBindingsWithRole(numBindings, numSubjectsPerBinding, nil)
}

// GetMultipleK8sRoleBindingsWithRole returns given number of roleBindings, each with given number of subjects and a
// reference to one of the given roles.
// The cluster role property will toggle from false to true and the referenced role will be a round robin of all given
// roles.
func GetMultipleK8sRoleBindingsWithRole(numBindings, numSubjectsPerBinding int,
	roles []*storage.K8SRole) []*storage.K8SRoleBinding {
	clusterRole := true
	bindings := make([]*storage.K8SRoleBinding, 0, numBindings)
	var roleIndex int
	var roleID string
	for i := 0; i < numBindings; i++ {
		name := fmt.Sprintf("k8sRoleBinding%d", i)
		clusterName := fmt.Sprintf("cluster%d", i)
		namespace := fmt.Sprintf("namespace%d", i)
		binding := GetK8sRoleBindingWithSubjects(
			uuid.NewV4().String(),
			name,
			uuid.NewV4().String(),
			clusterName,
			namespace,
			clusterRole,
			numSubjectsPerBinding)
		roleID, roleIndex = getRoleAndNewIndex(roleIndex, roles)
		binding.SetRoleId(roleID)
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

func getRoleAndNewIndex(i int, roles []*storage.K8SRole) (string, int) {
	if len(roles) == 0 {
		return "", 0
	}
	id := roles[i].GetId()
	if len(roles) == i-1 {
		return id, 0
	}
	return id, i + 1
}
