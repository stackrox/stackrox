package rbac

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	v1 "k8s.io/api/rbac/v1"
)

var (
	log = logging.LoggerForModule()
)

type storeImpl struct {
	lock                  sync.RWMutex
	hasBuiltInitialBucket bool
	syncedFlag            *concurrency.Flag

	roles              map[namespacedRoleRef]*storage.K8SRole
	bindingsByID       map[string]*storage.K8SRoleBinding
	bindingIDToRoleRef map[string]namespacedRoleRef
	roleRefToBindings  map[namespacedRoleRef]map[string]*storage.K8SRoleBinding

	bucketEvaluator *bucketEvaluator
}

type namespacedRoleRef struct {
	roleRef   v1.RoleRef
	namespace string
}

func (rs *storeImpl) GetPermissionLevelForDeployment(d NamespacedServiceAccount) storage.PermissionLevel {
	subject := &storage.Subject{
		Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
		Name:      d.GetServiceAccount(),
		Namespace: d.GetNamespace(),
	}

	rs.lock.Lock()
	defer rs.lock.Unlock()

	if !rs.hasBuiltInitialBucket {
		rs.hasBuiltInitialBucket = rs.rebuildEvaluatorBucketsNoLock()
		if !rs.hasBuiltInitialBucket {
			utils.Should(errors.New("deployment permissions should not be evaluated if rbac has not been synced"))
		}
	}

	return rs.bucketEvaluator.getBucketNoLock(subject)
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

func (rs *storeImpl) rebuildEvaluatorBucketsNoLock() bool {
	if !rs.syncedFlag.Get() {
		return false
	}
	roles := make([]*storage.K8SRole, 0, len(rs.roles))
	for _, r := range rs.roles {
		roles = append(roles, r)
	}
	bindings := make([]*storage.K8SRoleBinding, 0, len(rs.bindingsByID))
	for _, b := range rs.bindingsByID {
		bindings = append(bindings, b)
	}
	rs.bucketEvaluator = newBucketEvaluator(roles, bindings)
	return true
}

func (rs *storeImpl) updateBindingNoLock(roleID string, ref namespacedRoleRef, binding *storage.K8SRoleBinding) {
	newBinding := binding.Clone()
	newBinding.RoleId = roleID
	rs.bindingsByID[newBinding.GetId()] = newBinding
	rs.roleRefToBindings[ref][newBinding.GetId()] = newBinding
}

func (rs *storeImpl) upsertRoleGenericNoLock(ref namespacedRoleRef, role *storage.K8SRole) *storage.K8SRole {
	defer rs.rebuildEvaluatorBucketsNoLock()

	// Clone the role
	role = role.Clone()

	rs.roles[ref] = role

	// Find all the bindings that are registered for this namespacedRoleRef and assign their roleID
	for _, binding := range rs.roleRefToBindings[ref] {
		rs.updateBindingNoLock(role.GetId(), ref, binding)
	}

	return role
}

func (rs *storeImpl) removeRoleGenericNoLock(ref namespacedRoleRef, role *storage.K8SRole) *storage.K8SRole {
	defer rs.rebuildEvaluatorBucketsNoLock()

	delete(rs.roles, ref)
	// Find all the bindings that are registered for this namespacedRoleRef and remove their role ID as the reference is now broken
	for _, binding := range rs.roleRefToBindings[ref] {
		rs.updateBindingNoLock("", ref, binding)
	}
	return role
}

func (rs *storeImpl) upsertRoleBindingGenericNoLock(ref namespacedRoleRef, binding *storage.K8SRoleBinding) *storage.K8SRoleBinding {
	defer rs.rebuildEvaluatorBucketsNoLock()

	// If this update has made it so that the binding points at a new ref, then this will clean up the old ref
	if oldRef, oldRefExists := rs.bindingIDToRoleRef[binding.GetId()]; oldRefExists {
		rs.removeBindingFromMapsNoLock(oldRef, binding)
	}

	binding.RoleId = rs.roles[ref].GetId()

	rs.bindingsByID[binding.GetId()] = binding

	// This is used to track the potential cleanup seen above
	rs.bindingIDToRoleRef[binding.GetId()] = ref

	// This is used to track the ref to the role binding so when a role is inserted or removed, then we can update the binding
	if rs.roleRefToBindings[ref] == nil {
		rs.roleRefToBindings[ref] = make(map[string]*storage.K8SRoleBinding)
	}
	rs.roleRefToBindings[ref][binding.GetId()] = binding

	return binding
}

func (rs *storeImpl) removeRoleBindingGenericNoLock(ref namespacedRoleRef, binding *storage.K8SRoleBinding) *storage.K8SRoleBinding {
	defer rs.rebuildEvaluatorBucketsNoLock()

	binding.RoleId = rs.roles[ref].GetId()
	// Add or Replace the old binding if necessary.
	rs.removeBindingFromMapsNoLock(ref, binding)

	return binding
}

// Bookkeeping helper that removes a binding from the maps.
func (rs *storeImpl) removeBindingFromMapsNoLock(ref namespacedRoleRef, roxBinding *storage.K8SRoleBinding) {
	delete(rs.bindingsByID, roxBinding.GetId())
	delete(rs.bindingIDToRoleRef, roxBinding.GetId())
	delete(rs.roleRefToBindings[ref], roxBinding.GetId())
}
