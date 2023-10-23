package rbac

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/reconcile"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/deduper"
	"github.com/stackrox/rox/sensor/common/rbac"
	v1 "k8s.io/api/rbac/v1"
)

var (
	log = logging.LoggerForModule()
)

type reconcilePair struct {
	resID   string
	resType string
}

func (r reconcilePair) GetPair() (string, string) {
	return r.resID, r.resType
}

type storeImpl struct {
	lock sync.RWMutex

	roles    map[namespacedRoleRef]namespacedRole
	bindings map[namespacedBindingID]*namespacedBinding

	bucketEvaluator *evaluator
	dirty           bool
}

// Cleanup deletes all entries from store
func (rs *storeImpl) Cleanup() {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	rs.dirty = false
	rs.bucketEvaluator = newBucketEvaluator(nil, nil)
	rs.roles = make(map[namespacedRoleRef]namespacedRole)
	rs.bindings = make(map[namespacedBindingID]*namespacedBinding)
}

// ReconcileDelete is called after Sensor reconnects with Central and receives its state hashes.
// Reconciliation ensures that Sensor and Central have the same state by checking whether a given resource
// shall be deleted from Central.
func (rs *storeImpl) ReconcileDelete(resType, resID string, _ uint64) ([]reconcile.Resource, error) {
	if resType == deduper.TypeRole.String() {
		for _, role := range rs.roles {
			if role.latestUID == resID {
				return nil, nil
			}
		}
		return []reconcile.Resource{
			&reconcilePair{
				resID:   resID,
				resType: resType,
			},
		}, nil
	} else if resType == deduper.TypeBinding.String() {
		for _, binding := range rs.bindings {
			if binding.bindingID == resID {
				return nil, nil
			}
		}
		return []reconcile.Resource{
			&reconcilePair{
				resID:   resID,
				resType: resType,
			},
		}, nil
	}
	return nil, errors.Errorf("resource type %s not supported", resType)
}

func (rs *storeImpl) GetPermissionLevelForDeployment(d rbac.NamespacedServiceAccount) storage.PermissionLevel {
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
	namespacedBinding, _ := roleBindingToNamespacedBinding(binding)
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

func (rs *storeImpl) FindBindingForNamespacedRole(namespace, roleName string) []namespacedBindingID {
	rs.lock.RLock()
	defer rs.lock.RUnlock()

	var matched []namespacedBindingID
	for binding, ref := range rs.bindings {
		// During binding processing `ref.roleRef.namespace` will be set to "" if binding references a ClusterRole.
		// `namespace` parameter is also set to "" here, meaning that if the binding stored references a ClusterRole
		// we can determine such by checking if ref.roleRef.namespace == namespace. Otherwise, this simply matches that
		// a RoleBinding is in the same namespace that the Role being matched against.
		if ref.roleRef.name == roleName && ref.roleRef.namespace == namespace {
			matched = append(matched, binding)
		}
	}

	return matched
}

func (rs *storeImpl) FindSubjectForBindingID(namespace, name, uuid string) []namespacedSubject {
	rs.lock.RLock()
	defer rs.lock.RUnlock()

	id := namespacedBindingID{namespace: namespace, name: name, uid: uuid}
	if binding, ok := rs.bindings[id]; ok {
		return binding.subjects
	}
	return nil
}

func (rs *storeImpl) FindSubjectForRole(namespace, roleName string) []namespacedSubject {
	rs.lock.RLock()
	defer rs.lock.RUnlock()

	var matched []namespacedSubject
	for _, binding := range rs.bindings {
		if binding.roleRef.name == roleName && binding.roleRef.namespace == namespace {
			matched = append(matched, binding.subjects...)
		}
	}

	return matched
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
