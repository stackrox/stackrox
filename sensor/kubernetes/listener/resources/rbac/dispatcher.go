package rbac

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/utils"
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
			return toRoleEvent(r.store.RemoveRole(obj), action)
		}
		return toRoleEvent(r.store.UpsertRole(obj), action)
	case *v1.RoleBinding:
		if action == central.ResourceAction_REMOVE_RESOURCE {
			return toBindingEvent(r.store.RemoveBinding(obj), action)
		}
		return toBindingEvent(r.store.UpsertBinding(obj), action)
	case *v1.ClusterRole:
		if action == central.ResourceAction_REMOVE_RESOURCE {
			return toRoleEvent(r.store.RemoveClusterRole(obj), action)
		}
		return toRoleEvent(r.store.UpsertClusterRole(obj), action)
	case *v1.ClusterRoleBinding:
		if action == central.ResourceAction_REMOVE_RESOURCE {
			return toBindingEvent(r.store.RemoveClusterBinding(obj), action)
		}
		return toBindingEvent(r.store.UpsertClusterBinding(obj), action)
	}
	return nil
}
