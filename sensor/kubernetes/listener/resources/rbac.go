package resources

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	v1 "k8s.io/api/rbac/v1"
)

// roleDispatcher handles k8s role resource events.
type roleDispatcher struct {
	handler *rbacHandler
}

// newRoleDispatcher creates and returns a new role handler.
func newRoleDispatcher(handler *rbacHandler) *roleDispatcher {
	return &roleDispatcher{
		handler: handler,
	}
}

// Process processes a k8s role resource event, and returns the sensor events to emit in response.
func (r *roleDispatcher) ProcessEvent(obj interface{}, action central.ResourceAction) []*central.SensorEvent {

	switch obj := obj.(type) {
	case *v1.Role:
		return r.handler.processRoleEvents(obj, action)
	case *v1.ClusterRole:
		return r.handler.processClusterRoleEvents(obj, action)
	case *v1.RoleBinding:
		return r.handler.processRoleBindingEvents(obj, action)
	case *v1.ClusterRoleBinding:
		return r.handler.processClusterRoleBindingEvents(obj, action)
	}

	return nil
}

// clusterRoleDispatcher handles k8s clusterrole resource events.
type clusterRoleDispatcher struct {
	handler *rbacHandler
}

// newClusterRoleDispatcher creates and returns a new clusterrole handler.
func newClusterRoleDispatcher(handler *rbacHandler) *clusterRoleDispatcher {
	return &clusterRoleDispatcher{
		handler: handler,
	}
}

// Process processes a clusterrole resource event, and returns the sensor events to emit in response.
func (r *clusterRoleDispatcher) ProcessEvent(obj interface{}, action central.ResourceAction) []*central.SensorEvent {
	return r.handler.processClusterRoleEvents(obj.(*v1.ClusterRole), action)
}

// roleBindingDispatcher handles k8s rolebinding resource events.
type roleBindingDispatcher struct {
	handler *rbacHandler
}

// newRoleBindingDispatcher creates and returns a new rolebinding handler.
func newRoleBindingDispatcher(handler *rbacHandler) *roleBindingDispatcher {
	return &roleBindingDispatcher{
		handler: handler,
	}
}

// Process processes a clusterrolebinding resource event, and returns the sensor events to emit in response.
func (r *roleBindingDispatcher) ProcessEvent(obj interface{}, action central.ResourceAction) []*central.SensorEvent {
	return r.handler.processRoleBindingEvents(obj.(*v1.RoleBinding), action)
}

// clusterRoleBindingDispatcher handles k8s clusterrolebinding resource events.
type clusterRoleBindingDispatcher struct {
	handler *rbacHandler
}

// newClusterRoleBindingDispatcher creates and returns a new clusterrolebinding handler.
func newClusterRoleBindingDispatcher(handler *rbacHandler) *clusterRoleBindingDispatcher {
	return &clusterRoleBindingDispatcher{
		handler: handler,
	}
}

// Process processes a clusterrolebinding resource event, and returns the sensor events to emit in response.
func (r *clusterRoleBindingDispatcher) ProcessEvent(obj interface{}, action central.ResourceAction) []*central.SensorEvent {
	return r.handler.processClusterRoleBindingEvents(obj.(*v1.ClusterRoleBinding), action)
}

// rbacHandler handles rbac resource events and does the actual processing.
type rbacHandler struct {
	rbacStore *rbacStore
}

// rbacHandler handles rbac resource events and does the actual processing.
func newRBACHandler(store *rbacStore) *rbacHandler {
	return &rbacHandler{
		rbacStore: store,
	}
}

func (h *rbacHandler) processRoleEvents(role *v1.Role, action central.ResourceAction) []*central.SensorEvent {
	k8sRole := h.convertRole(role)
	return h.processRoleEventsWithType(k8sRole, action)
}

func (h *rbacHandler) processClusterRoleEvents(clusterRole *v1.ClusterRole, action central.ResourceAction) []*central.SensorEvent {
	k8sRole := h.convertClusterRole(clusterRole)
	return h.processRoleEventsWithType(k8sRole, action)
}

func (h *rbacHandler) processRoleBindingEvents(roleBinding *v1.RoleBinding, action central.ResourceAction) []*central.SensorEvent {
	k8sRoleBinding := h.convertRoleBinding(roleBinding)
	return h.processRoleBindingEventsWithType(k8sRoleBinding, roleBinding.RoleRef.Name, action)
}

func (h *rbacHandler) processClusterRoleBindingEvents(clusterRoleBinding *v1.ClusterRoleBinding, action central.ResourceAction) []*central.SensorEvent {
	k8sRoleBinding := h.convertClusterRoleBinding(clusterRoleBinding)
	return h.processRoleBindingEventsWithType(k8sRoleBinding, clusterRoleBinding.RoleRef.Name, action)
}

func (h *rbacHandler) processRoleEventsWithType(k8sRole *storage.K8SRole, action central.ResourceAction) []*central.SensorEvent {
	var events []*central.SensorEvent

	switch action {
	case central.ResourceAction_CREATE_RESOURCE:
		h.rbacStore.addOrUpdateRole(k8sRole)
		events = append(events, h.getRoleBindingEvents(k8sRole)...)
		h.rbacStore.removeBindingsForRoleName(k8sRole.GetNamespace(), k8sRole.GetName(), k8sRole.ClusterScope)

	case central.ResourceAction_UPDATE_RESOURCE:
		h.rbacStore.addOrUpdateRole(k8sRole)

	case central.ResourceAction_REMOVE_RESOURCE:
		h.rbacStore.removeRole(k8sRole)
	}

	events = append(events, &central.SensorEvent{
		Id:     k8sRole.GetId(),
		Action: action,
		Resource: &central.SensorEvent_Role{
			Role: k8sRole,
		},
	})

	return events
}

func (h *rbacHandler) processRoleBindingEventsWithType(k8sRoleBinding *storage.K8SRoleBinding, roleName string, action central.ResourceAction) []*central.SensorEvent {

	switch action {
	case central.ResourceAction_CREATE_RESOURCE:
	case central.ResourceAction_UPDATE_RESOURCE:
		if k8sRoleBinding.RoleId == "" {
			h.rbacStore.addOrUpdateRoleBinding(k8sRoleBinding, roleName)
		}

	case central.ResourceAction_REMOVE_RESOURCE:
		if k8sRoleBinding.RoleId == "" {
			h.rbacStore.removeRoleBinding(k8sRoleBinding, roleName)
		}
	}

	return []*central.SensorEvent{
		{
			Id:     k8sRoleBinding.GetId(),
			Action: action,
			Resource: &central.SensorEvent_Binding{
				Binding: k8sRoleBinding,
			},
		},
	}
}

func (h *rbacHandler) convertRole(role *v1.Role) *storage.K8SRole {
	return &storage.K8SRole{
		Id:           string(role.GetUID()),
		Name:         role.GetName(),
		Namespace:    role.GetNamespace(),
		ClusterName:  role.GetClusterName(),
		ClusterScope: false,
		CreatedAt:    protoconv.ConvertTimeToTimestamp(role.GetCreationTimestamp().Time),
		Rules:        getPolicyRules(role.Rules),
	}
}

func (h *rbacHandler) convertClusterRole(role *v1.ClusterRole) *storage.K8SRole {
	return &storage.K8SRole{
		Id:           string(role.GetUID()),
		Name:         role.GetName(),
		Namespace:    role.GetNamespace(),
		ClusterName:  role.GetClusterName(),
		ClusterScope: true,
		CreatedAt:    protoconv.ConvertTimeToTimestamp(role.GetCreationTimestamp().Time),
		Rules:        getPolicyRules(role.Rules),
	}
}

func (h *rbacHandler) convertRoleBinding(roleBinding *v1.RoleBinding) *storage.K8SRoleBinding {
	return &storage.K8SRoleBinding{
		Id:           string(roleBinding.GetUID()),
		Name:         roleBinding.GetName(),
		Namespace:    roleBinding.GetNamespace(),
		ClusterName:  roleBinding.GetClusterName(),
		ClusterScope: false,
		CreatedAt:    protoconv.ConvertTimeToTimestamp(roleBinding.GetCreationTimestamp().Time),
		Subjects:     getSubjects(roleBinding.Subjects),
		RoleId:       h.rbacStore.getRole(roleBinding.RoleRef.Name, roleBinding.GetNamespace()),
	}
}

func (h *rbacHandler) convertClusterRoleBinding(clusterRoleBinding *v1.ClusterRoleBinding) *storage.K8SRoleBinding {
	return &storage.K8SRoleBinding{
		Id:           string(clusterRoleBinding.GetUID()),
		Name:         clusterRoleBinding.GetName(),
		Namespace:    clusterRoleBinding.GetNamespace(),
		ClusterName:  clusterRoleBinding.GetClusterName(),
		ClusterScope: true,
		CreatedAt:    protoconv.ConvertTimeToTimestamp(clusterRoleBinding.GetCreationTimestamp().Time),
		Subjects:     getSubjects(clusterRoleBinding.Subjects),
		RoleId:       h.rbacStore.getRole(clusterRoleBinding.RoleRef.Name, clusterRoleBinding.GetNamespace()),
	}
}

func getPolicyRules(k8sRules []v1.PolicyRule) []*storage.PolicyRule {
	var rules []*storage.PolicyRule
	for _, rule := range k8sRules {
		rules = append(rules, &storage.PolicyRule{
			Verbs:     rule.Verbs,
			Resources: rule.Resources,
			ApiGroups: rule.APIGroups,
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

func (h *rbacHandler) getRoleBindingEvents(role *storage.K8SRole) (events []*central.SensorEvent) {
	bindings := h.rbacStore.getBindingsForRole(role)
	for _, binding := range bindings {
		binding.RoleId = role.GetId()
		events = append(events, &central.SensorEvent{
			Id:     string(binding.GetId()),
			Action: central.ResourceAction_UPDATE_RESOURCE,
			Resource: &central.SensorEvent_Binding{
				Binding: binding,
			},
		})
	}
	return
}
