package resources

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sync"
	v1 "k8s.io/api/rbac/v1"
)

// rbacUpdater handles correlating updates to K8s rbac types and generates events from them.
type rbacUpdater interface {
	upsertRole(role *v1.Role) *central.SensorEvent
	removeRole(role *v1.Role) *central.SensorEvent

	upsertClusterRole(role *v1.ClusterRole) *central.SensorEvent
	removeClusterRole(role *v1.ClusterRole) *central.SensorEvent

	upsertBinding(binding *v1.RoleBinding) *central.SensorEvent
	removeBinding(binding *v1.RoleBinding) *central.SensorEvent

	upsertClusterBinding(binding *v1.ClusterRoleBinding) *central.SensorEvent
	removeClusterBinding(binding *v1.ClusterRoleBinding) *central.SensorEvent

	assignPermissionLevelToDeployment(wrap *deploymentWrap)
}

type namespacedRoleRef struct {
	roleRef   v1.RoleRef
	namespace string
}

func newRBACUpdater() rbacUpdater {
	return &rbacUpdaterImpl{
		roles: make(map[namespacedRoleRef]*storage.K8SRole),

		bindingsByID:       make(map[string]*storage.K8SRoleBinding),
		bindingIDToRoleRef: make(map[string]namespacedRoleRef),
		roleRefToBindings:  make(map[namespacedRoleRef]map[string]*storage.K8SRoleBinding),

		// Incredibly unlikely that there are no roles and no bindings, but for safety initialize empty buckets
		bucketEvaluator: newBucketEvaluator(nil, nil),
	}
}

type rbacUpdaterImpl struct {
	lock sync.RWMutex

	roles              map[namespacedRoleRef]*storage.K8SRole
	bindingsByID       map[string]*storage.K8SRoleBinding
	bindingIDToRoleRef map[string]namespacedRoleRef
	roleRefToBindings  map[namespacedRoleRef]map[string]*storage.K8SRoleBinding

	bucketEvaluator *bucketEvaluator
}

func (rs *rbacUpdaterImpl) rebuildEvaluatorBucketsNoLock() {
	roles := make([]*storage.K8SRole, 0, len(rs.roles))
	for _, r := range rs.roles {
		roles = append(roles, r)
	}
	bindings := make([]*storage.K8SRoleBinding, 0, len(rs.bindingsByID))
	for _, b := range rs.bindingsByID {
		bindings = append(bindings, b)
	}
	rs.bucketEvaluator = newBucketEvaluator(roles, bindings)
}

func (rs *rbacUpdaterImpl) updateBindingNoLock(roleID string, ref namespacedRoleRef, binding *storage.K8SRoleBinding) {
	newBinding := proto.Clone(binding).(*storage.K8SRoleBinding)
	newBinding.RoleId = roleID
	rs.bindingsByID[newBinding.GetId()] = newBinding
	rs.roleRefToBindings[ref][newBinding.GetId()] = newBinding
}

func (rs *rbacUpdaterImpl) upsertRoleGenericNoLock(ref namespacedRoleRef, role *storage.K8SRole) *central.SensorEvent {
	defer rs.rebuildEvaluatorBucketsNoLock()

	// Clone the role
	role = proto.Clone(role).(*storage.K8SRole)

	rs.roles[ref] = role

	// Find all the bindings that are registered for this namespacedRoleRef and assign their roleID
	for _, binding := range rs.roleRefToBindings[ref] {
		rs.updateBindingNoLock(role.GetId(), ref, binding)
	}

	return toRoleEvent(role, central.ResourceAction_UPDATE_RESOURCE)
}

func (rs *rbacUpdaterImpl) removeRoleGenericNoLock(ref namespacedRoleRef, role *storage.K8SRole) *central.SensorEvent {
	defer rs.rebuildEvaluatorBucketsNoLock()

	delete(rs.roles, ref)
	// Find all the bindings that are registered for this namespacedRoleRef and remove their role ID as the reference is now broken
	for _, binding := range rs.roleRefToBindings[ref] {
		rs.updateBindingNoLock("", ref, binding)
	}
	return toRoleEvent(role, central.ResourceAction_REMOVE_RESOURCE)
}

func (rs *rbacUpdaterImpl) upsertRoleBindingGenericNoLock(ref namespacedRoleRef, binding *storage.K8SRoleBinding) *central.SensorEvent {
	defer rs.rebuildEvaluatorBucketsNoLock()

	// If this update has made it so that the binding points at a new ref, then this will clean up the old ref
	if oldRef, oldRefExists := rs.bindingIDToRoleRef[binding.GetId()]; oldRefExists {
		rs.removeBindingFromMaps(oldRef, binding)
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

	return toBindingEvent(binding, central.ResourceAction_UPDATE_RESOURCE)
}

func (rs *rbacUpdaterImpl) removeRoleBindingGenericNoLock(ref namespacedRoleRef, binding *storage.K8SRoleBinding) *central.SensorEvent {
	defer rs.rebuildEvaluatorBucketsNoLock()

	binding.RoleId = rs.roles[ref].GetId()
	// Add or Replace the old binding if necessary.
	rs.removeBindingFromMaps(ref, binding)

	return toBindingEvent(binding, central.ResourceAction_REMOVE_RESOURCE)
}

func (rs *rbacUpdaterImpl) upsertRole(role *v1.Role) *central.SensorEvent {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	return rs.upsertRoleGenericNoLock(roleAsRef(role), toRoxRole(role))
}

func (rs *rbacUpdaterImpl) removeRole(role *v1.Role) *central.SensorEvent {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	return rs.removeRoleGenericNoLock(roleAsRef(role), toRoxRole(role))
}

func (rs *rbacUpdaterImpl) upsertClusterRole(role *v1.ClusterRole) *central.SensorEvent {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	return rs.upsertRoleGenericNoLock(clusterRoleAsRef(role), toRoxClusterRole(role))
}

func (rs *rbacUpdaterImpl) removeClusterRole(role *v1.ClusterRole) *central.SensorEvent {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	return rs.removeRoleGenericNoLock(clusterRoleAsRef(role), toRoxClusterRole(role))
}

func (rs *rbacUpdaterImpl) upsertBinding(binding *v1.RoleBinding) *central.SensorEvent {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	return rs.upsertRoleBindingGenericNoLock(roleBindingRefToNamespaceRef(binding), toRoxRoleBinding(binding))
}

func (rs *rbacUpdaterImpl) removeBinding(binding *v1.RoleBinding) *central.SensorEvent {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	return rs.removeRoleBindingGenericNoLock(roleBindingRefToNamespaceRef(binding), toRoxRoleBinding(binding))
}

func (rs *rbacUpdaterImpl) upsertClusterBinding(binding *v1.ClusterRoleBinding) *central.SensorEvent {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	return rs.upsertRoleBindingGenericNoLock(clusterRoleBindingRefToNamespaceRef(binding), toRoxClusterRoleBinding(binding))
}

func (rs *rbacUpdaterImpl) removeClusterBinding(binding *v1.ClusterRoleBinding) *central.SensorEvent {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	return rs.removeRoleBindingGenericNoLock(clusterRoleBindingRefToNamespaceRef(binding), toRoxClusterRoleBinding(binding))
}

// Bookkeeping helper that removes a binding from the maps.
func (rs *rbacUpdaterImpl) removeBindingFromMaps(ref namespacedRoleRef, roxBinding *storage.K8SRoleBinding) {
	delete(rs.bindingsByID, roxBinding.GetId())
	delete(rs.bindingIDToRoleRef, roxBinding.GetId())
	delete(rs.roleRefToBindings[ref], roxBinding.GetId())
}

// Static conversion functions.
///////////////////////////////

func toRoxRole(role *v1.Role) *storage.K8SRole {
	return &storage.K8SRole{
		Id:          string(role.GetUID()),
		Name:        role.GetName(),
		Namespace:   role.GetNamespace(),
		ClusterName: role.GetClusterName(),
		Labels:      role.GetLabels(),
		Annotations: role.GetAnnotations(),
		ClusterRole: false,
		CreatedAt:   protoconv.ConvertTimeToTimestamp(role.GetCreationTimestamp().Time),
		Rules:       getPolicyRules(role.Rules),
	}
}

func toRoxClusterRole(role *v1.ClusterRole) *storage.K8SRole {
	return &storage.K8SRole{
		Id:          string(role.GetUID()),
		Name:        role.GetName(),
		Namespace:   role.GetNamespace(),
		ClusterName: role.GetClusterName(),
		Labels:      role.GetLabels(),
		Annotations: role.GetAnnotations(),
		ClusterRole: true,
		CreatedAt:   protoconv.ConvertTimeToTimestamp(role.GetCreationTimestamp().Time),
		Rules:       getPolicyRules(role.Rules),
	}
}

func toRoxRoleBinding(roleBinding *v1.RoleBinding) *storage.K8SRoleBinding {
	return &storage.K8SRoleBinding{
		Id:          string(roleBinding.GetUID()),
		Name:        roleBinding.GetName(),
		Namespace:   roleBinding.GetNamespace(),
		ClusterName: roleBinding.GetClusterName(),
		Labels:      roleBinding.GetLabels(),
		Annotations: roleBinding.GetAnnotations(),
		ClusterRole: false,
		CreatedAt:   protoconv.ConvertTimeToTimestamp(roleBinding.GetCreationTimestamp().Time),
		Subjects:    getSubjects(roleBinding.Subjects),
	}
}

func toRoxClusterRoleBinding(clusterRoleBinding *v1.ClusterRoleBinding) *storage.K8SRoleBinding {
	return &storage.K8SRoleBinding{
		Id:          string(clusterRoleBinding.GetUID()),
		Name:        clusterRoleBinding.GetName(),
		Namespace:   clusterRoleBinding.GetNamespace(),
		ClusterName: clusterRoleBinding.GetClusterName(),
		Labels:      clusterRoleBinding.GetLabels(),
		Annotations: clusterRoleBinding.GetAnnotations(),
		ClusterRole: true,
		CreatedAt:   protoconv.ConvertTimeToTimestamp(clusterRoleBinding.GetCreationTimestamp().Time),
		Subjects:    getSubjects(clusterRoleBinding.Subjects),
	}
}

func getPolicyRules(k8sRules []v1.PolicyRule) []*storage.PolicyRule {
	rules := make([]*storage.PolicyRule, 0, len(k8sRules))
	for _, rule := range k8sRules {
		rules = append(rules, &storage.PolicyRule{
			Verbs:           rule.Verbs,
			Resources:       rule.Resources,
			ApiGroups:       rule.APIGroups,
			ResourceNames:   rule.ResourceNames,
			NonResourceUrls: rule.NonResourceURLs,
		})
	}
	return rules
}

func getSubjectKind(kind string) storage.SubjectKind {
	switch kind {
	case v1.ServiceAccountKind:
		return storage.SubjectKind_SERVICE_ACCOUNT
	case v1.UserKind:
		return storage.SubjectKind_USER
	case v1.GroupKind:
		return storage.SubjectKind_GROUP
	default:
		log.Warnf("unexpected subject kind %s", kind)
		return storage.SubjectKind_SERVICE_ACCOUNT
	}
}

func getSubjects(k8sSubjects []v1.Subject) []*storage.Subject {
	subjects := make([]*storage.Subject, 0, len(k8sSubjects))
	for _, subject := range k8sSubjects {
		subjects = append(subjects, &storage.Subject{
			Kind:      getSubjectKind(subject.Kind),
			Name:      subject.Name,
			Namespace: subject.Namespace,
		})
	}
	return subjects
}

// Static event construction.
/////////////////////////////

func toRoleEvent(role *storage.K8SRole, action central.ResourceAction) *central.SensorEvent {
	return &central.SensorEvent{
		Id:     role.GetId(),
		Action: action,
		Resource: &central.SensorEvent_Role{
			Role: role,
		},
	}
}

func toBindingEvent(binding *storage.K8SRoleBinding, action central.ResourceAction) *central.SensorEvent {
	return &central.SensorEvent{
		Id:     binding.GetId(),
		Action: action,
		Resource: &central.SensorEvent_Binding{
			Binding: binding,
		},
	}
}

// K8s helpers since roles don't have their own refs (eye-roll).
////////////////////////////////////////////////////////////////

func roleAsRef(role *v1.Role) namespacedRoleRef {
	return namespacedRoleRef{
		roleRef: v1.RoleRef{
			Kind:     "Role",
			Name:     role.GetName(),
			APIGroup: "rbac.authorization.k8s.io",
		},
		namespace: role.GetNamespace(),
	}
}

func clusterRoleAsRef(role *v1.ClusterRole) namespacedRoleRef {
	return namespacedRoleRef{
		roleRef: v1.RoleRef{
			Kind:     "ClusterRole",
			Name:     role.GetName(),
			APIGroup: "rbac.authorization.k8s.io",
		},
		namespace: "",
	}
}

func roleBindingRefToNamespaceRef(rolebinding *v1.RoleBinding) namespacedRoleRef {
	if rolebinding.RoleRef.Kind == "ClusterRole" {
		return namespacedRoleRef{
			roleRef:   rolebinding.RoleRef,
			namespace: "",
		}
	}

	return namespacedRoleRef{
		roleRef:   rolebinding.RoleRef,
		namespace: rolebinding.GetNamespace(),
	}
}

func clusterRoleBindingRefToNamespaceRef(rolebinding *v1.ClusterRoleBinding) namespacedRoleRef {
	return namespacedRoleRef{
		roleRef:   rolebinding.RoleRef,
		namespace: "",
	}
}
