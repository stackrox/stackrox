package resources

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sync"
	v1 "k8s.io/api/rbac/v1"
)

// rbacUpdater handles correlating updates to K8s rbac types and generates events from them.
type rbacUpdater interface {
	upsertRole(role *v1.Role) []*central.SensorEvent
	removeRole(role *v1.Role) []*central.SensorEvent

	upsertClusterRole(role *v1.ClusterRole) []*central.SensorEvent
	removeClusterRole(role *v1.ClusterRole) []*central.SensorEvent

	upsertBinding(binding *v1.RoleBinding) []*central.SensorEvent
	removeBinding(binding *v1.RoleBinding) []*central.SensorEvent

	upsertClusterBinding(binding *v1.ClusterRoleBinding) []*central.SensorEvent
	removeClusterBinding(binding *v1.ClusterRoleBinding) []*central.SensorEvent
}

func newRBACUpdater() rbacUpdater {
	return &rbacUpdaterImpl{
		roles:              make(map[v1.RoleRef]*storage.K8SRole),
		bindingsByID:       make(map[string]*storage.K8SRoleBinding),
		bindingIDToRoleRef: make(map[string]v1.RoleRef),
	}
}

type rbacUpdaterImpl struct {
	lock sync.Mutex

	roles              map[v1.RoleRef]*storage.K8SRole
	bindingsByID       map[string]*storage.K8SRoleBinding
	bindingIDToRoleRef map[string]v1.RoleRef
}

func (rs *rbacUpdaterImpl) upsertRole(role *v1.Role) (events []*central.SensorEvent) {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	ref := roleAsRef(role)
	roxRole := toRoxRole(role)
	events = append(events, toRoleEvent(roxRole, central.ResourceAction_UPDATE_RESOURCE))

	_, exists := rs.roles[ref]
	rs.roles[ref] = roxRole

	// Reassign all role bindings to have the new id if it has changed.
	if !exists {
		for bindingID, bindingRef := range rs.bindingIDToRoleRef {
			if bindingRef == ref {
				if binding, bindingExists := rs.bindingsByID[bindingID]; bindingExists && binding.RoleId != roxRole.GetId() {
					binding.RoleId = roxRole.GetId()
					events = append(events, toBindingEvent(binding, central.ResourceAction_UPDATE_RESOURCE))
				}
			}
		}
	}
	return
}

func (rs *rbacUpdaterImpl) removeRole(role *v1.Role) (events []*central.SensorEvent) {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	ref := roleAsRef(role)
	roxRole := toRoxRole(role)
	events = append(events, toRoleEvent(roxRole, central.ResourceAction_REMOVE_RESOURCE))

	_, exists := rs.roles[ref]

	// Reassign all assigned bindings to have no role id.
	if exists {
		delete(rs.roles, ref)
		for bindingID, bindingRef := range rs.bindingIDToRoleRef {
			if bindingRef == ref {
				if binding, bindingExists := rs.bindingsByID[bindingID]; bindingExists {
					binding.RoleId = ""
					events = append(events, toBindingEvent(binding, central.ResourceAction_UPDATE_RESOURCE))
				}
			}
		}
	}
	return
}

func (rs *rbacUpdaterImpl) upsertClusterRole(role *v1.ClusterRole) (events []*central.SensorEvent) {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	ref := clusterRoleAsRef(role)
	roxRole := toRoxClusterRole(role)
	events = append(events, toRoleEvent(roxRole, central.ResourceAction_UPDATE_RESOURCE))

	_, exists := rs.roles[ref]
	rs.roles[ref] = roxRole

	// Reassign all role bindings to have the new id if it has changed.
	if !exists {
		for bindingID, bindingRef := range rs.bindingIDToRoleRef {
			if bindingRef == ref {
				if binding, bindingExists := rs.bindingsByID[bindingID]; bindingExists && binding.RoleId != roxRole.GetId() {
					binding.RoleId = roxRole.GetId()
					events = append(events, toBindingEvent(binding, central.ResourceAction_UPDATE_RESOURCE))
				}
			}
		}
	}
	return
}

func (rs *rbacUpdaterImpl) removeClusterRole(role *v1.ClusterRole) (events []*central.SensorEvent) {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	ref := clusterRoleAsRef(role)
	roxRole := toRoxClusterRole(role)
	events = append(events, toRoleEvent(roxRole, central.ResourceAction_REMOVE_RESOURCE))

	_, exists := rs.roles[ref]

	// Reassign all assigned bindings to have no role id.
	if exists {
		delete(rs.roles, ref)
		for bindingID, bindingRef := range rs.bindingIDToRoleRef {
			if bindingRef == ref {
				if binding, bindingExists := rs.bindingsByID[bindingID]; bindingExists {
					binding.RoleId = ""
					events = append(events, toBindingEvent(binding, central.ResourceAction_UPDATE_RESOURCE))
				}
			}
		}
	}
	return
}

func (rs *rbacUpdaterImpl) upsertBinding(binding *v1.RoleBinding) (events []*central.SensorEvent) {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	// Check for an existing matching role.
	currentRole, roleExists := rs.roles[binding.RoleRef]

	// Convert to rox version of role binding
	var roxBinding *storage.K8SRoleBinding
	if roleExists {
		roxBinding = toRoxRoleBinding(currentRole.GetId(), binding)
	} else {
		roxBinding = toRoxRoleBinding("", binding)
	}
	events = append(events, toBindingEvent(roxBinding, central.ResourceAction_UPDATE_RESOURCE))

	// Add or Replace the old binding if necessary.
	rs.addBindingToMaps(binding.RoleRef, roxBinding)
	return
}

func (rs *rbacUpdaterImpl) removeBinding(binding *v1.RoleBinding) (events []*central.SensorEvent) {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	// Check for an existing matching role.
	currentRole, roleExists := rs.roles[binding.RoleRef]

	// Convert to rox version of role binding
	var roxBinding *storage.K8SRoleBinding
	if !roleExists {
		roxBinding = toRoxRoleBinding(currentRole.GetId(), binding)
	} else {
		roxBinding = toRoxRoleBinding("", binding)
	}
	events = append(events, toBindingEvent(roxBinding, central.ResourceAction_REMOVE_RESOURCE))

	// Add or Replace the old binding if necessary.
	rs.removeBindingFromMaps(binding.RoleRef, roxBinding)
	return
}

func (rs *rbacUpdaterImpl) upsertClusterBinding(binding *v1.ClusterRoleBinding) (events []*central.SensorEvent) {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	// Check for an existing matching role.
	currentRole, roleExists := rs.roles[binding.RoleRef]

	// Convert to rox version of role binding
	var roxBinding *storage.K8SRoleBinding
	if roleExists {
		roxBinding = toRoxClusterRoleBinding(currentRole.GetId(), binding)
	} else {
		roxBinding = toRoxClusterRoleBinding("", binding)
	}
	events = append(events, toBindingEvent(roxBinding, central.ResourceAction_UPDATE_RESOURCE))

	// Add or Replace the old binding if necessary.
	rs.addBindingToMaps(binding.RoleRef, roxBinding)
	return
}

func (rs *rbacUpdaterImpl) removeClusterBinding(binding *v1.ClusterRoleBinding) (events []*central.SensorEvent) {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	// Check for an existing matching role.
	currentRole, roleExists := rs.roles[binding.RoleRef]

	// Convert to rox version of role binding
	var roxBinding *storage.K8SRoleBinding
	if !roleExists {
		roxBinding = toRoxClusterRoleBinding(currentRole.GetId(), binding)
	} else {
		roxBinding = toRoxClusterRoleBinding("", binding)
	}
	events = append(events, toBindingEvent(roxBinding, central.ResourceAction_REMOVE_RESOURCE))

	// Add or Replace the old binding if necessary.
	rs.removeBindingFromMaps(binding.RoleRef, roxBinding)
	return
}

// Bookkeeping helper that adds/updates a binding in the maps.
func (rs *rbacUpdaterImpl) addBindingToMaps(ref v1.RoleRef, roxBinding *storage.K8SRoleBinding) bool {
	if oldRef, oldRefExists := rs.bindingIDToRoleRef[roxBinding.GetId()]; oldRefExists {
		rs.removeBindingFromMaps(oldRef, roxBinding) // remove binding for previous role ref
	}

	_, bindingExists := rs.bindingsByID[roxBinding.GetId()]
	rs.bindingsByID[roxBinding.GetId()] = roxBinding
	rs.bindingIDToRoleRef[roxBinding.GetId()] = ref
	return !bindingExists
}

// Bookkeeping helper that removes a binding from the maps.
func (rs *rbacUpdaterImpl) removeBindingFromMaps(ref v1.RoleRef, roxBinding *storage.K8SRoleBinding) bool {
	_, removed := rs.bindingsByID[roxBinding.GetId()]
	if removed {
		delete(rs.bindingsByID, roxBinding.GetId())
		delete(rs.bindingIDToRoleRef, roxBinding.GetId())
	}
	return removed
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

func toRoxRoleBinding(roleID string, roleBinding *v1.RoleBinding) *storage.K8SRoleBinding {
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
		RoleId:      roleID,
	}
}

func toRoxClusterRoleBinding(roleID string, clusterRoleBinding *v1.ClusterRoleBinding) *storage.K8SRoleBinding {
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
		RoleId:      roleID,
	}
}

func getPolicyRules(k8sRules []v1.PolicyRule) []*storage.PolicyRule {
	var rules []*storage.PolicyRule
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
	var subjects []*storage.Subject
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
		Id:     string(role.GetId()),
		Action: action,
		Resource: &central.SensorEvent_Role{
			Role: role,
		},
	}
}

func toBindingEvent(binding *storage.K8SRoleBinding, action central.ResourceAction) *central.SensorEvent {
	return &central.SensorEvent{
		Id:     string(binding.GetId()),
		Action: action,
		Resource: &central.SensorEvent_Binding{
			Binding: binding,
		},
	}
}

// K8s helpers since roles don't have their own refs (eye-roll).
////////////////////////////////////////////////////////////////

func roleAsRef(role *v1.Role) v1.RoleRef {
	return v1.RoleRef{
		Kind:     "Role",
		Name:     role.GetName(),
		APIGroup: "rbac.authorization.k8s.io",
	}
}

func clusterRoleAsRef(role *v1.ClusterRole) v1.RoleRef {
	return v1.RoleRef{
		Kind:     "ClusterRole",
		Name:     role.GetName(),
		APIGroup: "rbac.authorization.k8s.io",
	}
}
