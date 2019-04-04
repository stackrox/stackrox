package resources

import (
	"github.com/stackrox/rox/generated/storage"
)

type rbacStore struct {
	roles                map[string]map[string]*storage.K8SRole
	roleToBinding        map[string]map[string][]*storage.K8SRoleBinding
	clusterRoleToBinding map[string]map[string][]*storage.K8SRoleBinding
}

// newServiceStore creates and returns a new service store.
func newRBACStore() *rbacStore {
	return &rbacStore{
		roles:                make(map[string]map[string]*storage.K8SRole),
		roleToBinding:        make(map[string]map[string][]*storage.K8SRoleBinding),
		clusterRoleToBinding: make(map[string]map[string][]*storage.K8SRoleBinding),
	}
}

func (rs *rbacStore) addOrUpdateRole(role *storage.K8SRole) {
	nsMap := rs.roles[role.GetNamespace()]
	if nsMap == nil {
		nsMap = make(map[string]*storage.K8SRole)
		rs.roles[role.GetNamespace()] = nsMap
	}
	nsMap[role.GetId()] = role
}

func (rs *rbacStore) removeRole(role *storage.K8SRole) {
	nsMap := rs.roles[role.GetNamespace()]
	if nsMap == nil {
		return
	}
	delete(nsMap, role.GetId())
}

func (rs *rbacStore) getRole(roleName string, namespace string) string {
	roleMap := rs.roles[namespace]

	if roleMap == nil {
		return ""
	}

	for id, role := range roleMap {
		if role.GetName() == roleName {
			return id
		}
	}

	return ""
}

func (rs *rbacStore) getBindingsForRole(role *storage.K8SRole) []*storage.K8SRoleBinding {
	var rolebindingsMap map[string][]*storage.K8SRoleBinding

	roleName := role.GetName()
	namespace := role.GetNamespace()

	if role.ClusterScope {
		rolebindingsMap = rs.clusterRoleToBinding[namespace]
	} else {
		rolebindingsMap = rs.roleToBinding[namespace]
	}
	return rolebindingsMap[roleName]
}

func (rs *rbacStore) addOrUpdateRoleBinding(roleBinding *storage.K8SRoleBinding, roleName string) {

	var roleBindingMap map[string][]*storage.K8SRoleBinding
	if roleBinding.ClusterScope {
		roleBindingMap = rs.clusterRoleToBinding[roleBinding.GetNamespace()]
	} else {
		roleBindingMap = rs.roleToBinding[roleBinding.GetNamespace()]

	}

	if roleBindingMap == nil {
		roleBindingMap = make(map[string][]*storage.K8SRoleBinding)

		if roleBinding.ClusterScope {
			rs.clusterRoleToBinding[roleBinding.GetNamespace()] = roleBindingMap
		} else {
			rs.roleToBinding[roleBinding.GetNamespace()] = roleBindingMap

		}
	}

	roleBindingMap[roleName] = append(roleBindingMap[roleName], roleBinding)
}

func (rs *rbacStore) removeRoleBinding(roleBinding *storage.K8SRoleBinding, roleName string) {
	var roleBindingMap map[string][]*storage.K8SRoleBinding

	if roleBinding.ClusterScope {
		roleBindingMap = rs.clusterRoleToBinding[roleBinding.GetNamespace()]
	} else {
		roleBindingMap = rs.roleToBinding[roleBinding.GetNamespace()]

	}

	if roleBindingMap == nil {
		return
	}

	bindings := roleBindingMap[roleName]
	for i, binding := range bindings {
		if binding.GetId() == roleBinding.GetId() {
			roleBindingMap[roleName] = append(bindings[:i], bindings[i+1:]...)
			break
		}
	}
}

func (rs *rbacStore) removeBindingsForRoleName(namespace string, roleName string, clusterScope bool) {
	if clusterScope {
		clusterRoleBindingsMap := rs.clusterRoleToBinding[namespace]
		if clusterRoleBindingsMap != nil {
			delete(clusterRoleBindingsMap, roleName)
		}
		return
	}

	roleBindingMap := rs.roleToBinding[namespace]
	if roleBindingMap != nil {
		delete(roleBindingMap, roleName)
	}

}
