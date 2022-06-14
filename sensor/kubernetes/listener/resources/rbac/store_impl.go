package rbac

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	v1 "k8s.io/api/rbac/v1"
)

var (
	log = logging.LoggerForModule()
)

type storeImpl struct {
	lock sync.RWMutex

	roles    map[namespacedRoleRef]namespacedRole
	bindings map[namespacedBindingID]*namespacedBinding

	bucketEvaluator *evaluator
	dirty           bool
}

func (rs *storeImpl) GetPermissionLevelForDeployment(d NamespacedServiceAccount) storage.PermissionLevel {
	subject := &storage.Subject{
		Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
		Name:      d.GetServiceAccount(),
		Namespace: d.GetNamespace(),
	}

	rs.lock.Lock()
	defer rs.lock.Unlock()

	if rs.dirty {
		rs.rebuildEvaluatorBucketsNoLock()
	}

	return rs.bucketEvaluator.GetPermissionLevelForSubject(subject)
}

func (rs *storeImpl) GetNamespacedRoleIDOrEmpty(roleRef namespacedRoleRef) string {
	rs.lock.Lock()
	defer rs.lock.Unlock()
	role, ok := rs.roles[roleRef]
	if ok {
		return role.latestUID
	}
	return ""
}

func (rs *storeImpl) UpsertRole(role *v1.Role) {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	rs.upsertRoleGenericNoLock(roleAsRef(role), roleAsNamespacedRole(role))
}

func (rs *storeImpl) RemoveRole(role *v1.Role) {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	rs.removeRoleGenericNoLock(roleAsRef(role))
}

func (rs *storeImpl) UpsertClusterRole(role *v1.ClusterRole) {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	rs.upsertRoleGenericNoLock(clusterRoleAsRef(role), clusterRoleAsNamespacedRole(role))
}

func (rs *storeImpl) RemoveClusterRole(role *v1.ClusterRole) {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	rs.removeRoleGenericNoLock(clusterRoleAsRef(role))
}

func (rs *storeImpl) UpsertBinding(binding *v1.RoleBinding) {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	bindingID := roleBindingToNamespacedBindingID(binding)
	namespacedBinding := roleBindingToNamespacedBinding(binding)
	rs.upsertRoleBindingGenericNoLock(bindingID, namespacedBinding)
}

func (rs *storeImpl) RemoveBinding(binding *v1.RoleBinding) {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	bindingID := roleBindingToNamespacedBindingID(binding)
	rs.removeRoleBindingGenericNoLock(bindingID)
}

func (rs *storeImpl) UpsertClusterBinding(binding *v1.ClusterRoleBinding) {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	bindingID := clusterRoleBindingToNamespacedBindingID(binding)
	namespacedBinding := clusterRoleBindingToNamespacedBinding(binding)
	rs.upsertRoleBindingGenericNoLock(bindingID, namespacedBinding)
}

func (rs *storeImpl) RemoveClusterBinding(binding *v1.ClusterRoleBinding) {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	bindingID := clusterRoleBindingToNamespacedBindingID(binding)
	rs.removeRoleBindingGenericNoLock(bindingID)
}

func (rs *storeImpl) rebuildEvaluatorBucketsNoLock() {
	rs.bucketEvaluator = newBucketEvaluator(rs.roles, rs.bindings)
	rs.dirty = false
}

func (rs *storeImpl) upsertRoleGenericNoLock(ref namespacedRoleRef, role namespacedRole) {
	oldRole, oldRoleExists := rs.roles[ref]
	if oldRoleExists && role == oldRole {
		return
	}
	rs.roles[ref] = role
	rs.markDirtyNoLock() // All related bindings now refer to the new role.
}

func (rs *storeImpl) removeRoleGenericNoLock(ref namespacedRoleRef) {
	delete(rs.roles, ref)
	rs.markDirtyNoLock() // All related bindings now refer to no concrete role.
}

func (rs *storeImpl) upsertRoleBindingGenericNoLock(bindingID namespacedBindingID, binding *namespacedBinding) {
	oldBinding, oldBindingExists := rs.bindings[bindingID]
	if oldBindingExists && binding.Equal(oldBinding) {
		return
	}

	rs.bindings[bindingID] = binding
	rs.markDirtyNoLock()
}

func (rs *storeImpl) removeRoleBindingGenericNoLock(bindingID namespacedBindingID) {
	delete(rs.bindings, bindingID)
	rs.markDirtyNoLock()
}

func (rs *storeImpl) markDirtyNoLock() {
	rs.dirty = true
}
