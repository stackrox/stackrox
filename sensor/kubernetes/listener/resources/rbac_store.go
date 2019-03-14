package resources

import (
	"github.com/stackrox/rox/generated/storage"
)

type rbacStore struct {
	roles                map[string]map[string]*storage.K8SRole
	roleToBinding        map[string]map[string]*storage.K8SRoleBinding
	clusterRoleToBinding map[string]map[string]*storage.K8SRoleBinding
}

// newServiceStore creates and returns a new service store.
func newRBACStore() *rbacStore {
	return &rbacStore{
		roles:                make(map[string]map[string]*storage.K8SRole),
		roleToBinding:        make(map[string]map[string]*storage.K8SRoleBinding),
		clusterRoleToBinding: make(map[string]map[string]*storage.K8SRoleBinding),
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

func (rs *rbacStore) getBindingsForRole(roleName string, clusterRole bool) []*storage.K8SRoleBinding {
	var rolebindingsMap map[string]*storage.K8SRoleBinding

	if clusterRole {
		rolebindingsMap = rs.clusterRoleToBinding[roleName]
	}
	rolebindingsMap = rs.roleToBinding[roleName]

	var roleBindings []*storage.K8SRoleBinding
	for _, binding := range rolebindingsMap {
		roleBindings = append(roleBindings, binding)
	}

	return roleBindings
}

func (rs *rbacStore) addOrUpdateRoleBinding(roleBinding *storage.K8SRoleBinding) {
	var rolebindingsMap map[string]*storage.K8SRoleBinding

	if roleBinding.ClusterScope {
		rolebindingsMap = rs.clusterRoleToBinding[roleBinding.GetRoleId()]
	}
	rolebindingsMap = rs.roleToBinding[roleBinding.GetRoleId()]

	if rolebindingsMap == nil {
		rolebindingsMap = make(map[string]*storage.K8SRoleBinding)
	}
	rolebindingsMap[roleBinding.GetRoleId()] = roleBinding
}

func (rs *rbacStore) removeRoleBinding(roleBinding *storage.K8SRoleBinding) {
	var rolebindingsMap map[string]*storage.K8SRoleBinding

	if roleBinding.ClusterScope {
		rolebindingsMap = rs.clusterRoleToBinding[roleBinding.GetRoleId()]
	}
	rolebindingsMap = rs.roleToBinding[roleBinding.GetRoleId()]

	if rolebindingsMap != nil {
		delete(rolebindingsMap, roleBinding.GetId())
	}
}
