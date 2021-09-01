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

	roles              map[namespacedRoleRef]*storage.K8SRole
	bindingsByID       map[string]*storage.K8SRoleBinding
	bindingIDToRoleRef map[string]namespacedRoleRef
	roleRefToBindings  map[namespacedRoleRef]map[string]*storage.K8SRoleBinding

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

func (rs *storeImpl) UpsertRole(role *v1.Role) *storage.K8SRole {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	return rs.upsertRoleGenericNoLock(roleAsRef(role), toRoxRole(role))
}

func (rs *storeImpl) RemoveRole(role *v1.Role) *storage.K8SRole {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	return rs.removeRoleGenericNoLock(roleAsRef(role), toRoxRole(role))
}

func (rs *storeImpl) UpsertClusterRole(role *v1.ClusterRole) *storage.K8SRole {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	return rs.upsertRoleGenericNoLock(clusterRoleAsRef(role), toRoxClusterRole(role))
}

func (rs *storeImpl) RemoveClusterRole(role *v1.ClusterRole) *storage.K8SRole {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	return rs.removeRoleGenericNoLock(clusterRoleAsRef(role), toRoxClusterRole(role))
}

func (rs *storeImpl) UpsertBinding(binding *v1.RoleBinding) *storage.K8SRoleBinding {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	return rs.upsertRoleBindingGenericNoLock(roleBindingRefToNamespaceRef(binding), toRoxRoleBinding(binding))
}

func (rs *storeImpl) RemoveBinding(binding *v1.RoleBinding) *storage.K8SRoleBinding {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	return rs.removeRoleBindingGenericNoLock(roleBindingRefToNamespaceRef(binding), toRoxRoleBinding(binding))
}

func (rs *storeImpl) UpsertClusterBinding(binding *v1.ClusterRoleBinding) *storage.K8SRoleBinding {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	return rs.upsertRoleBindingGenericNoLock(clusterRoleBindingRefToNamespaceRef(binding), toRoxClusterRoleBinding(binding))
}

func (rs *storeImpl) RemoveClusterBinding(binding *v1.ClusterRoleBinding) *storage.K8SRoleBinding {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	return rs.removeRoleBindingGenericNoLock(clusterRoleBindingRefToNamespaceRef(binding), toRoxClusterRoleBinding(binding))
}

func (rs *storeImpl) rebuildEvaluatorBucketsNoLock() {
	rs.bucketEvaluator = newBucketEvaluator(rs.roles, rs.bindingsByID)
	rs.dirty = false
}

func (rs *storeImpl) updateBindingNoLock(roleID string, ref namespacedRoleRef, binding *storage.K8SRoleBinding) {
	newBinding := binding.Clone()
	newBinding.RoleId = roleID
	rs.bindingsByID[newBinding.GetId()] = newBinding
	rs.roleRefToBindings[ref][newBinding.GetId()] = newBinding
	rs.markDirtyNoLock()
}

func (rs *storeImpl) upsertRoleGenericNoLock(ref namespacedRoleRef, role *storage.K8SRole) *storage.K8SRole {
	// Clone the role
	role = role.Clone()

	// We do not need to check if role was changed since we are not syncing roles.
	// Every role upsert changes the state (ROX-7837).

	rs.roles[ref] = role

	// Find all the bindings that are registered for this namespacedRoleRef and assign their roleID
	for _, binding := range rs.roleRefToBindings[ref] {
		rs.updateBindingNoLock(role.GetId(), ref, binding)
	}

	return role
}

func (rs *storeImpl) removeRoleGenericNoLock(ref namespacedRoleRef, role *storage.K8SRole) *storage.K8SRole {
	delete(rs.roles, ref)
	// Find all the bindings that are registered for this namespacedRoleRef and remove their role ID as the reference is now broken
	for _, binding := range rs.roleRefToBindings[ref] {
		rs.updateBindingNoLock("", ref, binding)
	}
	return role
}

func (rs *storeImpl) upsertRoleBindingGenericNoLock(ref namespacedRoleRef, binding *storage.K8SRoleBinding) *storage.K8SRoleBinding {
	binding.RoleId = rs.roles[ref].GetId()
	if binding.Equal(rs.bindingsByID[binding.GetId()]) {
		return binding
	}

	// If this update has made it so that the binding points at a new ref, then this will clean up the old ref
	if oldRef, oldRefExists := rs.bindingIDToRoleRef[binding.GetId()]; oldRefExists {
		rs.removeBindingFromMapsNoLock(oldRef, binding)
	}

	rs.bindingsByID[binding.GetId()] = binding

	// This is used to track the potential cleanup seen above
	rs.bindingIDToRoleRef[binding.GetId()] = ref

	// This is used to track the ref to the role binding so when a role is inserted or removed, then we can update the binding
	if rs.roleRefToBindings[ref] == nil {
		rs.roleRefToBindings[ref] = make(map[string]*storage.K8SRoleBinding)
	}
	rs.roleRefToBindings[ref][binding.GetId()] = binding
	rs.markDirtyNoLock()
	return binding
}

func (rs *storeImpl) removeRoleBindingGenericNoLock(ref namespacedRoleRef, binding *storage.K8SRoleBinding) *storage.K8SRoleBinding {
	binding.RoleId = rs.roles[ref].GetId()
	// Add or Replace the old binding if necessary.
	rs.removeBindingFromMapsNoLock(ref, binding)
	rs.markDirtyNoLock()
	return binding
}

// Bookkeeping helper that removes a binding from the maps.
func (rs *storeImpl) removeBindingFromMapsNoLock(ref namespacedRoleRef, roxBinding *storage.K8SRoleBinding) {
	delete(rs.bindingsByID, roxBinding.GetId())
	delete(rs.bindingIDToRoleRef, roxBinding.GetId())
	delete(rs.roleRefToBindings[ref], roxBinding.GetId())
}

func (rs *storeImpl) markDirtyNoLock() {
	rs.dirty = true
}
