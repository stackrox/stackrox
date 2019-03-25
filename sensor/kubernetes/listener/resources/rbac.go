package resources

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"k8s.io/api/rbac/v1"
)

// rbacDispatcher handles rbac resource events.
type rbacDispatcher struct {
	handler *rbacHandler
}

// newRBACDispatcher creates and returns a new rbac handler.
func newRBACDispatcher(handler *rbacHandler) *rbacDispatcher {
	return &rbacDispatcher{
		handler: handler,
	}
}

// Process processes a rbac resource event, and returns the sensor events to emit in response.
func (r *rbacDispatcher) ProcessEvent(obj interface{}, action central.ResourceAction) []*central.SensorEvent {

	switch obj := obj.(type) {
	case *v1.Role:
		r.handler.processRoleEvents(obj, action)
	case *v1.ClusterRole:
		r.handler.processClusterRoleEvents(obj, action)
	case *v1.RoleBinding:
		r.handler.processRoleBindingEvents(obj, action)
	case *v1.ClusterRoleBinding:
		r.handler.processClusterRoleBindingEvents(obj, action)
	}

	return nil
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
	return h.processRoleBindingEventsWithType(k8sRoleBinding, action)
}

func (h *rbacHandler) processClusterRoleBindingEvents(clusterRoleBinding *v1.ClusterRoleBinding, action central.ResourceAction) []*central.SensorEvent {
	k8sRoleBinding := h.convertClusterRoleBinding(clusterRoleBinding)
	return h.processRoleBindingEventsWithType(k8sRoleBinding, action)
}

func (h *rbacHandler) processRoleEventsWithType(k8sRole *storage.K8SRole, action central.ResourceAction) []*central.SensorEvent {
	var events []*central.SensorEvent

	switch action {
	case central.ResourceAction_CREATE_RESOURCE:
		h.rbacStore.addOrUpdateRole(k8sRole)
		h.updateRoleBindingEvents(k8sRole, events)

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

func (h *rbacHandler) processRoleBindingEventsWithType(k8sRoleBinding *storage.K8SRoleBinding, action central.ResourceAction) []*central.SensorEvent {

	switch action {
	case central.ResourceAction_CREATE_RESOURCE:
	case central.ResourceAction_UPDATE_RESOURCE:
		h.rbacStore.addOrUpdateRoleBinding(k8sRoleBinding)

	case central.ResourceAction_REMOVE_RESOURCE:
		h.rbacStore.removeRoleBinding(k8sRoleBinding)
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

func (h *rbacHandler) updateRoleBindingEvents(role *storage.K8SRole, events []*central.SensorEvent) {
	bindings := h.rbacStore.getBindingsForRole(role.GetName(), role.GetClusterScope())
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
}
