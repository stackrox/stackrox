package rbac

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/common/store/resolver"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
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
func (r *Dispatcher) ProcessEvent(obj, _ interface{}, action central.ResourceAction) *component.ResourceEvent {
	rbacUpdate := r.processEvent(obj, action)
	if rbacUpdate.event == nil {
		utils.Should(errors.Errorf("rbac obj %+v was not correlated to a sensor event", obj))
		return nil
	}
	events := []*central.SensorEvent{
		rbacUpdate.event,
	}
	componentMessage := component.NewResourceEvent(events, nil, nil)

	component.MergeResourceEvents(componentMessage, component.NewDeploymentRefEvent(
		resolver.ResolveDeploymentsByMultipleServiceAccounts(mapReference(rbacUpdate.deploymentReference)),
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
	case *v1.RoleBinding:
		update.deploymentReference.AddAll(r.store.FindSubjectForBindingID(obj.GetNamespace(), string(obj.GetUID()))...)
		if action == central.ResourceAction_REMOVE_RESOURCE {
			r.store.RemoveBinding(obj)
		} else {
			r.store.UpsertBinding(obj)
		}
		// This is appended again in case the binding changed, and now it should match a different service account
		update.deploymentReference.AddAll(r.store.FindSubjectForBindingID(obj.GetNamespace(), string(obj.GetUID()))...)
		update.event = toBindingEvent(r.toRoxBinding(obj), action)
	case *v1.ClusterRole:
		if action == central.ResourceAction_REMOVE_RESOURCE {
			r.store.RemoveClusterRole(obj)
		} else {
			r.store.UpsertClusterRole(obj)
		}
		update.deploymentReference.AddAll(r.store.FindSubjectForRole(obj.GetNamespace(), obj.GetName())...)
		update.event = toRoleEvent(toRoxClusterRole(obj), action)
	case *v1.ClusterRoleBinding:
		update.deploymentReference.AddAll(r.store.FindSubjectForBindingID(obj.GetNamespace(), string(obj.GetUID()))...)
		if action == central.ResourceAction_REMOVE_RESOURCE {
			r.store.RemoveClusterBinding(obj)
		} else {
			r.store.UpsertClusterBinding(obj)
		}
		// This is appended again in case the binding changed, and now it should match a different service account
		update.deploymentReference.AddAll(r.store.FindSubjectForBindingID(obj.GetNamespace(), string(obj.GetUID()))...)
		update.event = toBindingEvent(r.toRoxClusterRoleBinding(obj), action)
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
