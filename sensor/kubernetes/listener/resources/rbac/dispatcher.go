package rbac

import (
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/utils"
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

// ProcessEvent handles RBAC-related events
func (r *Dispatcher) ProcessEvent(obj, _ interface{}, action central.ResourceAction) []*central.SensorEvent {
	evt := r.processEvent(obj, action)
	if evt == nil {
		utils.Should(errors.Errorf("rbac obj %+v was not correlated to a sensor event", obj))
		return nil
	}
	return []*central.SensorEvent{
		evt,
	}
}

func (r *Dispatcher) processEvent(obj interface{}, action central.ResourceAction) *central.SensorEvent {
	switch obj := obj.(type) {
	case *v1.Role:
		if action == central.ResourceAction_REMOVE_RESOURCE {
			r.store.RemoveRole(obj)
		} else {
			r.store.UpsertRole(obj)
		}
		return toRoleEvent(toRoxRole(obj), action)
	case *v1.RoleBinding:
		if action == central.ResourceAction_REMOVE_RESOURCE {
			r.store.RemoveBinding(obj)
		} else {
			r.store.UpsertBinding(obj)
		}
		return toBindingEvent(r.toRoxBinding(obj), action)
	case *v1.ClusterRole:
		if action == central.ResourceAction_REMOVE_RESOURCE {
			r.store.RemoveClusterRole(obj)
		} else {
			r.store.UpsertClusterRole(obj)
		}
		return toRoleEvent(toRoxClusterRole(obj), action)
	case *v1.ClusterRoleBinding:
		if action == central.ResourceAction_REMOVE_RESOURCE {
			r.store.RemoveClusterBinding(obj)
		} else {
			r.store.UpsertClusterBinding(obj)
		}
		return toBindingEvent(r.toRoxClusterRoleBinding(obj), action)
	}
	return nil
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
