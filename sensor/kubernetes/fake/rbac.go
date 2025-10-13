package fake

import (
	"math/rand"

	"github.com/stackrox/rox/pkg/k8srbac"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	rolesPool          = make(map[string][]string)
	serviceAccountPool = make(map[string][]string)
)

func getRoleBinding(id string) *v1.RoleBinding {
	ns := namespacePool.mustGetRandomElem()
	possibleServiceAccounts := serviceAccountPool[ns]
	if len(possibleServiceAccounts) == 0 {
		return nil
	}
	possibleRoles := rolesPool[ns]
	if len(possibleRoles) == 0 {
		return nil
	}
	sa := possibleServiceAccounts[rand.Intn(len(possibleServiceAccounts))]
	return &v1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      randStringWithLength(16),
			Namespace: ns,
			UID:       idOrNewUID(id),
		},
		Subjects: []v1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      sa,
				Namespace: ns,
			},
			{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Group",
				Name:     randStringWithLength(16),
			},
			{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "User",
				Name:     randStringWithLength(16),
			},
		},
		RoleRef: v1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     possibleRoles[rand.Intn(len(possibleRoles))],
		},
	}
}

func getAPIGroups() []string {
	return []string{"", "*"}
}

func getResources() []string {
	return []string{
		"*",
		"configmaps",
		"events",
		"secrets",
	}
}

func getVerbs() []string {
	return k8srbac.ResourceVerbs.AsSlice()
}

func getRole(id string) *v1.Role {
	ns := namespacePool.mustGetRandomElem()
	role := &v1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      randStringWithLength(16),
			Namespace: ns,
			UID:       idOrNewUID(id),
		},
		Rules: []v1.PolicyRule{
			{
				Verbs:     getVerbs(),
				APIGroups: getAPIGroups(),
				Resources: getResources(),
			},
		},
	}
	rolesPool[ns] = append(rolesPool[ns], role.Name)
	return role
}

func getServiceAccount(id string) *corev1.ServiceAccount {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      randStringWithLength(16),
			Namespace: namespacePool.mustGetRandomElem(),
			UID:       idOrNewUID(id),
		},
	}
	serviceAccountPool[sa.Namespace] = append(serviceAccountPool[sa.Namespace], sa.Name)
	return sa
}

func (w *WorkloadManager) getRBAC(workload RBACWorkload, saIDs, roleIDs, rolebindingIDs []string) []runtime.Object {
	objects := make([]runtime.Object, 0, workload.NumServiceAccounts+workload.NumRoles+workload.NumBindings)
	for i := 0; i < workload.NumServiceAccounts; i++ {
		sa := getServiceAccount(getID(saIDs, i))
		w.writeID(serviceAccountPrefix, sa.UID)
		objects = append(objects, sa)
	}
	for i := 0; i < workload.NumRoles; i++ {
		role := getRole(getID(roleIDs, i))
		w.writeID(rolesPrefix, role.UID)
		objects = append(objects, role)
	}
	for i := 0; i < workload.NumBindings; i++ {
		if binding := getRoleBinding(getID(rolebindingIDs, i)); binding != nil {
			w.writeID(rolebindingsPrefix, binding.UID)
			objects = append(objects, binding)
		}
	}
	return objects
}
