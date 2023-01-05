package rbac

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	v1 "k8s.io/api/rbac/v1"
)

// Dispatcher handles RBAC-related events
type Dispatcher struct {
	store Store
	// pendingBindings holds the Binding events temporarily while the roles are not yet received. Any bindings without
	// a role does not influence in the `PermissionLevel` of a deployment, so holding the binding until a role that
	// matches it is received won't cause any loss of updates. This maps binding IDs to their K8s resource objects.
	pendingBindings map[string]*storage.K8SRoleBinding
}

// rbacUpdate represents an RBAC event with the reference to deployments that might require reprocessing. These
// deployments are dependents on this resource. The reference is based on the service account subject on the role
// binding. Multiple subjects can be returned since the role can be updated with a subject change.
type rbacUpdate struct {
	events              []*central.SensorEvent
	deploymentReference set.Set[namespacedSubject]
}

// NewDispatcher creates new instance of Dispatcher
func NewDispatcher(store Store) *Dispatcher {
	return &Dispatcher{
		store:           store,
		pendingBindings: map[string]*storage.K8SRoleBinding{},
	}
}

// ProcessEvent handles RBAC-related events
func (r *Dispatcher) ProcessEvent(obj, _ interface{}, action central.ResourceAction) *component.ResourceEvent {
	update := r.processEvent(obj, action)
	return component.NewResourceEvent(update.events, nil, nil)
}

func (r *Dispatcher) processEvent(obj interface{}, action central.ResourceAction) rbacUpdate {
	var update rbacUpdate
	switch obj := obj.(type) {
	case *v1.Role:
		if action == central.ResourceAction_REMOVE_RESOURCE {
			r.store.RemoveRole(obj)
		} else {
			r.store.UpsertRole(obj)
		}
		update.events = append(update.events, toRoleEvent(toRoxRole(obj), action))
		update.events = append(update.events, r.processPendingBindingsMatching(obj.GetNamespace(), obj.GetName(), string(obj.GetUID()), false)...)
	case *v1.RoleBinding:
		if action == central.ResourceAction_REMOVE_RESOURCE {
			r.store.RemoveBinding(obj)
		} else {
			r.store.UpsertBinding(obj)
		}
		roxBinding := r.toRoxBinding(obj)
		if roxBinding.GetRoleId() == "" {
			r.pendingBindings[roxBinding.GetId()] = roxBinding
		}
		update.events = append(update.events, toBindingEvent(roxBinding, action))
	case *v1.ClusterRole:
		if action == central.ResourceAction_REMOVE_RESOURCE {
			r.store.RemoveClusterRole(obj)
		} else {
			r.store.UpsertClusterRole(obj)
		}
		update.events = append(update.events, toRoleEvent(toRoxClusterRole(obj), action))
		update.events = append(update.events, r.processPendingBindingsMatching(obj.GetNamespace(), obj.GetName(), string(obj.GetUID()), true)...)
	case *v1.ClusterRoleBinding:
		if action == central.ResourceAction_REMOVE_RESOURCE {
			r.store.RemoveClusterBinding(obj)
		} else {
			r.store.UpsertClusterBinding(obj)
		}
		roxBinding := r.toRoxClusterRoleBinding(obj)
		if roxBinding.GetRoleId() == "" {
			r.pendingBindings[roxBinding.GetId()] = roxBinding
		}
		update.events = append(update.events, toBindingEvent(roxBinding, action))
	}
	return update
}

// processPendingBindingsMatching finds any binding events that were sent without a roleID, updates the roleID and
// send them to central. Pending Bindings are then removed from the internal map, as any new updates will be able
// to fetch the matching RoleID in the RBAC store.
func (r *Dispatcher) processPendingBindingsMatching(namespace, name, id string, clusterWide bool) []*central.SensorEvent {
	var updateEvents []*central.SensorEvent
	bindings := r.store.FindBindingIDForRole(namespace, name, clusterWide)
	log.Debugf("Found (%d) bindings for role (%s, %s): %+v", len(bindings), namespace, name, bindings)
	for _, binding := range bindings {
		if preProcessed, ok := r.pendingBindings[binding]; ok {
			preProcessed.RoleId = id
			updateEvents = append(updateEvents, toBindingEvent(preProcessed, central.ResourceAction_UPDATE_RESOURCE))
			delete(r.pendingBindings, binding)
		}
	}
	return updateEvents
}

func (r *Dispatcher) toRoxBinding(binding *v1.RoleBinding) *storage.K8SRoleBinding {
	namespacedBinding, isClusterRole := roleBindingToNamespacedBinding(binding)
	roleID := r.store.GetNamespacedRoleIDOrEmpty(namespacedBinding.roleRef)
	roxRoleBinding := toRoxRoleBinding(binding, roleID, isClusterRole)
	return roxRoleBinding
}

func (r *Dispatcher) toRoxClusterRoleBinding(binding *v1.ClusterRoleBinding) *storage.K8SRoleBinding {
	namespacedBinding := clusterRoleBindingToNamespacedBinding(binding)
	roleID := r.store.GetNamespacedRoleIDOrEmpty(namespacedBinding.roleRef)
	roxRoleBinding := toRoxClusterRoleBinding(binding, roleID)
	return roxRoleBinding
}
