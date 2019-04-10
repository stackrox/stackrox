package resources

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	v1 "k8s.io/api/rbac/v1"
)

//////////////////////
// Handle role events.
type roleDispatcher struct {
	store rbacUpdater
}

func newRoleDispatcher(store rbacUpdater) *roleDispatcher {
	return &roleDispatcher{
		store: store,
	}
}

func (r *roleDispatcher) ProcessEvent(obj interface{}, action central.ResourceAction) []*central.SensorEvent {
	switch obj := obj.(type) {
	case *v1.Role:
		if action == central.ResourceAction_REMOVE_RESOURCE {
			return r.store.removeRole(obj)
		}
		return r.store.upsertRole(obj)
	}
	return nil
}

//////////////////////////////
// Handle cluster role events.
type clusterRoleDispatcher struct {
	store rbacUpdater
}

func newClusterRoleDispatcher(store rbacUpdater) *clusterRoleDispatcher {
	return &clusterRoleDispatcher{
		store: store,
	}
}

func (r *clusterRoleDispatcher) ProcessEvent(obj interface{}, action central.ResourceAction) []*central.SensorEvent {
	switch obj := obj.(type) {
	case *v1.ClusterRole:
		if action == central.ResourceAction_REMOVE_RESOURCE {
			return r.store.removeClusterRole(obj)
		}
		return r.store.upsertClusterRole(obj)
	}
	return nil
}

//////////////////////////////
// Handle role binding events.
type bindingDispatcher struct {
	store rbacUpdater
}

func newBindingDispatcher(store rbacUpdater) *bindingDispatcher {
	return &bindingDispatcher{
		store: store,
	}
}

func (r *bindingDispatcher) ProcessEvent(obj interface{}, action central.ResourceAction) []*central.SensorEvent {
	switch obj := obj.(type) {
	case *v1.RoleBinding:
		if action == central.ResourceAction_REMOVE_RESOURCE {
			return r.store.removeBinding(obj)
		}
		return r.store.upsertBinding(obj)
	}
	return nil
}

//////////////////////////////////////
// Handle cluster role binding events.
type clusterBindingDispatcher struct {
	store rbacUpdater
}

func newClusterBindingDispatcher(store rbacUpdater) *clusterBindingDispatcher {
	return &clusterBindingDispatcher{
		store: store,
	}
}

func (r *clusterBindingDispatcher) ProcessEvent(obj interface{}, action central.ResourceAction) []*central.SensorEvent {
	switch obj := obj.(type) {
	case *v1.ClusterRoleBinding:
		if action == central.ResourceAction_REMOVE_RESOURCE {
			return r.store.removeClusterBinding(obj)
		}
		return r.store.upsertClusterBinding(obj)
	}
	return nil
}
