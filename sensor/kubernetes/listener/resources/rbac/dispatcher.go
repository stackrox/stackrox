package rbac

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/sensor/common/store/resolver"
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

// NewDispatcher creates new instance of Dispatcher
func NewDispatcher(store Store) *Dispatcher {
	return &Dispatcher{
		store:           store,
		pendingBindings: map[string]*storage.K8SRoleBinding{},
	}
}

// ProcessEvent handles RBAC-related events
func (r *Dispatcher) ProcessEvent(obj, _ interface{}, action central.ResourceAction) *component.ResourceEvent {
	rbacUpdate := r.processEvent(obj, action)
	events := rbacUpdate.forward
	if rbacUpdate.event != nil {
		events = append(events, rbacUpdate.event)
	}

	componentMessage := component.NewResourceEvent(events, nil, nil)

	serviceAccountReferences := mapReference(rbacUpdate.deploymentReference)
	component.MergeResourceEvents(componentMessage, component.NewDeploymentRefEvent(
		resolver.ResolveDeploymentsByMultipleServiceAccounts(serviceAccountReferences),
		central.ResourceAction_UPDATE_RESOURCE,
	))

	return componentMessage
}

func mapReference(subjects set.Set[namespacedSubject]) []resolver.NamespaceServiceAccount {
	var result []resolver.NamespaceServiceAccount
	for _, subj := range subjects.AsSlice() {
		namespace, serviceAccount, err := subj.decode()
		if err != nil {
			log.Errorf("failed to decode namespaced service account in RBAC in-memory store: %s", err)
		}
		result = append(result, resolver.NamespaceServiceAccount{Namespace: namespace, ServiceAccount: serviceAccount})
	}
	return result
}

// rbacUpdate represents an RBAC event with the reference to deployments that might require reprocessing. These
// deployments are dependents on this resource. The reference is based on the service account subject on the role
// binding. Multiple subjects can be returned since the role can be updated with a subject change.
type rbacUpdate struct {
	event               *central.SensorEvent
	forward             []*central.SensorEvent
	deploymentReference set.Set[namespacedSubject]
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
		update.event = toRoleEvent(toRoxRole(obj), action)
		update.deploymentReference.AddAll(r.store.FindSubjectForRole(obj.GetNamespace(), obj.GetName())...)
		update.forward = r.processPendingBindingsMatching(obj.GetNamespace(), obj.GetName(), string(obj.GetUID()))
	case *v1.RoleBinding:
		update.deploymentReference.AddAll(r.store.FindSubjectForBindingID(obj.GetNamespace(), string(obj.GetUID()))...)
		if action == central.ResourceAction_REMOVE_RESOURCE {
			r.store.RemoveBinding(obj)
		} else {
			r.store.UpsertBinding(obj)
		}
		// This is appended again in case the binding changed, and now it should match a different service account
		update.deploymentReference.AddAll(r.store.FindSubjectForBindingID(obj.GetNamespace(), string(obj.GetUID()))...)
		roxBinding := r.toRoxBinding(obj)
		if roxBinding.GetRoleId() == "" {
			// add this binding to pending binding list
			r.pendingBindings[roxBinding.GetId()] = roxBinding
		}
		update.event = toBindingEvent(roxBinding, action)
	case *v1.ClusterRole:
		if action == central.ResourceAction_REMOVE_RESOURCE {
			r.store.RemoveClusterRole(obj)
		} else {
			r.store.UpsertClusterRole(obj)
		}
		update.deploymentReference.AddAll(r.store.FindSubjectForRole(obj.GetNamespace(), obj.GetName())...)
		update.event = toRoleEvent(toRoxClusterRole(obj), action)
		update.forward = r.processPendingBindingsMatching(obj.GetNamespace(), obj.GetName(), string(obj.GetUID()))
	case *v1.ClusterRoleBinding:
		update.deploymentReference.AddAll(r.store.FindSubjectForBindingID(obj.GetNamespace(), string(obj.GetUID()))...)
		if action == central.ResourceAction_REMOVE_RESOURCE {
			r.store.RemoveClusterBinding(obj)
		} else {
			r.store.UpsertClusterBinding(obj)
		}
		// This is appended again in case the binding changed, and now it should match a different service account
		update.deploymentReference.AddAll(r.store.FindSubjectForBindingID(obj.GetNamespace(), string(obj.GetUID()))...)
		roxBinding := r.toRoxClusterRoleBinding(obj)
		if roxBinding.GetRoleId() == "" {
			// add this binding to pending binding list
			r.pendingBindings[roxBinding.GetId()] = roxBinding
		}
		update.event = toBindingEvent(roxBinding, action)
	}
	return update
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

func (r *Dispatcher) processPendingBindingsMatching(namespace, name, id string) []*central.SensorEvent {
	var updateEvents []*central.SensorEvent
	bindings := r.store.FindBindingIdForRole(namespace, name)
	log.Debugf("Found (%d) bindings for role (%s, %s): %+v", len(bindings), namespace, name, bindings)
	for _, binding := range bindings {
		if preProcessed, ok := r.pendingBindings[binding]; ok {
			// Update RoleId and send
			preProcessed.RoleId = id
			updateEvents = append(updateEvents, toBindingEvent(preProcessed, central.ResourceAction_UPDATE_RESOURCE))
			delete(r.pendingBindings, binding)
		}
	}
	return updateEvents
}
