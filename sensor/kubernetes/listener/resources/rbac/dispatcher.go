package rbac

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/message"
	v1 "k8s.io/api/rbac/v1"
)

// Dispatcher handles RBAC-related events
type Dispatcher struct {
	store Store
}

// NewDispatcher creates new instance of Dispatcher
func NewDispatcher(store Store) *Dispatcher {
	return &Dispatcher{
		store: store,
	}
}

type rbacUpdated struct {
	event           *central.SensorEvent
	topLevelUpdates []namespacedSubject
}

// ProcessEvent handles RBAC-related events
func (r *Dispatcher) ProcessEvent(obj, _ interface{}, action central.ResourceAction) *message.ResourceEvent {
	result := r.processEvent(obj, action)
	if result.event == nil {
		utils.Should(errors.Errorf("rbac obj %+v was not correlated to a sensor event", obj))
		return nil
	}
	events := []*central.SensorEvent{
		result.event,
	}
	return &message.ResourceEvent{
		ForwardMessages:                  events,
		CompatibilityDetectionDeployment: nil,
		ReprocessDeployments:             nil,
		// Deployments that must be updated
		DeploymentRefs: mapToDeploymentRef(result.topLevelUpdates),
	}
}

func mapToDeploymentRef(subjects []namespacedSubject) []message.DeploymentRef {
	var topLevelReferences []message.DeploymentRef
	for _, subj := range subjects {
		var ref message.DeploymentRef
		parts := strings.Split(string(subj), "#")
		if len(parts[0]) != 0 {
			ref.Namespace = parts[0]
		}
		ref.Subject = parts[1]
		topLevelReferences = append(topLevelReferences, ref)
	}
	return topLevelReferences
}

func (r *Dispatcher) processEvent(obj interface{}, action central.ResourceAction) rbacUpdated {
	var result rbacUpdated
	switch obj := obj.(type) {
	case *v1.Role:
		if action == central.ResourceAction_REMOVE_RESOURCE {
			r.store.RemoveRole(obj)
		} else {
			r.store.UpsertRole(obj)
		}
		result.topLevelUpdates = r.store.FindSubjectsFromNamespacedRole(obj.GetNamespace(), obj.GetName())
		result.event = toRoleEvent(toRoxRole(obj), action)
	case *v1.RoleBinding:
		// Try to find subjects before. Because if it's an update or delete, the bindings will be
		// overwritten. So they need to be merged together
		result.topLevelUpdates = r.store.FindSubjectForBinding(obj.GetNamespace(), string(obj.GetUID()))
		binding := roleBindingToNamespacedBinding(obj)
		result.topLevelUpdates = append(result.topLevelUpdates, binding.subjects...)

		if action == central.ResourceAction_REMOVE_RESOURCE {
			r.store.RemoveBinding(obj)
		} else {
			r.store.UpsertBinding(obj)
		}
		result.event = toBindingEvent(r.toRoxBinding(obj), action)
	case *v1.ClusterRole:
		if action == central.ResourceAction_REMOVE_RESOURCE {
			r.store.RemoveClusterRole(obj)
		} else {
			r.store.UpsertClusterRole(obj)
		}
		result.topLevelUpdates = r.store.FindSubjectsFromNamespacedRole("", obj.GetName())
		result.event = toRoleEvent(toRoxClusterRole(obj), action)
	case *v1.ClusterRoleBinding:
		// Try to find subjects before. Because if it's an update or delete, the bindings will be
		// overwritten. So they need to be merged together
		result.topLevelUpdates = r.store.FindSubjectForBinding(obj.GetNamespace(), string(obj.GetUID()))
		binding := clusterRoleBindingToNamespacedBinding(obj)
		result.topLevelUpdates = append(result.topLevelUpdates, binding.subjects...)

		if action == central.ResourceAction_REMOVE_RESOURCE {
			r.store.RemoveClusterBinding(obj)
		} else {
			r.store.UpsertClusterBinding(obj)
		}
		result.topLevelUpdates = r.store.FindSubjectForBinding("", string(obj.GetUID()))
		result.event = toBindingEvent(r.toRoxClusterRoleBinding(obj), action)
	}
	return result
}

func (r *Dispatcher) toRoxBinding(binding *v1.RoleBinding) *storage.K8SRoleBinding {
	namespacedBinding := roleBindingToNamespacedBinding(binding)
	roleID := r.store.GetNamespacedRoleIDOrEmpty(namespacedBinding.roleRef)
	roxRoleBinding := toRoxRoleBinding(binding, roleID)
	return roxRoleBinding
}

func (r *Dispatcher) toRoxClusterRoleBinding(binding *v1.ClusterRoleBinding) *storage.K8SRoleBinding {
	namespacedBinding := clusterRoleBindingToNamespacedBinding(binding)
	roleID := r.store.GetNamespacedRoleIDOrEmpty(namespacedBinding.roleRef)
	roxRoleBinding := toRoxClusterRoleBinding(binding, roleID)
	return roxRoleBinding
}
