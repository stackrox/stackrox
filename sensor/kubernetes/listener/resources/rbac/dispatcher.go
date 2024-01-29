package rbac

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/sensor/common/store/resolver"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Dispatcher handles RBAC-related events
type Dispatcher struct {
	store   Store
	fetcher *bindingFetcher
}

// rbacUpdate represents an RBAC event with the reference to deployments that might require reprocessing. These
// deployments are dependents on this resource. The reference is based on the service account subject on the role
// binding. Multiple subjects can be returned since the role can be updated with a subject change.
type rbacUpdate struct {
	events              []*central.SensorEvent
	deploymentReference set.Set[namespacedSubject]
}

// NewDispatcher creates new instance of Dispatcher
func NewDispatcher(store Store, k8sAPI kubernetes.Interface) *Dispatcher {
	return &Dispatcher{
		store:   store,
		fetcher: newBindingFetcher(k8sAPI),
	}
}

// ProcessEvent handles RBAC-related events
func (r *Dispatcher) ProcessEvent(obj, _ interface{}, action central.ResourceAction) *component.ResourceEvent {
	update := r.processEvent(obj, action)
	events := component.NewEvent(update.events...)
	serviceAccountReferences := mapReference(update.deploymentReference)
	events.AddDeploymentReference(resolver.ResolveDeploymentsByMultipleServiceAccounts(serviceAccountReferences))
	return events
}

func mapReference(subjects set.Set[namespacedSubject]) []resolver.NamespaceServiceAccount {
	var result []resolver.NamespaceServiceAccount
	for _, subj := range subjects.AsSlice() {
		namespace, serviceAccount, err := subj.splitNamespaceAndName()
		if err != nil {
			log.Errorf("failed to decode namespaced service account (%s) in RBAC in-memory store: %s", subj, err)
			continue
		}
		result = append(result, resolver.NamespaceServiceAccount{Namespace: namespace, ServiceAccount: serviceAccount})
	}
	return result
}

func (r *Dispatcher) processEvent(obj interface{}, action central.ResourceAction) rbacUpdate {
	var update rbacUpdate
	switch obj := obj.(type) {
	case *v1.Role:
		update.events = append(update.events, toRoleEvent(toRoxRole(obj), action))
		if action == central.ResourceAction_REMOVE_RESOURCE {
			r.store.RemoveRole(obj)
			update.events = append(update.events, r.mustGenerateRelatedEvents(obj, "", false)...)
		} else if action == central.ResourceAction_CREATE_RESOURCE || action == central.ResourceAction_SYNC_RESOURCE {
			r.store.UpsertRole(obj)
			// In case the role is being created, or it's during sensor startup, dependent bindings should be processed.
			update.events = append(update.events, r.mustGenerateRelatedEvents(obj, string(obj.GetUID()), false)...)
		} else if action == central.ResourceAction_UPDATE_RESOURCE {
			r.store.UpsertRole(obj)
		}
		update.deploymentReference.AddAll(r.findSubjectForRole(obj)...)
	case *v1.RoleBinding:
		update.deploymentReference.AddAll(r.findSubjectForBinding(obj)...)
		if action == central.ResourceAction_REMOVE_RESOURCE {
			r.store.RemoveBinding(obj)
		} else {
			r.store.UpsertBinding(obj)
		}
		roxBinding := r.toRoxBinding(obj)
		update.events = append(update.events, toBindingEvent(roxBinding, action))
		// Binding deployment reference has to be appended twice in case the binding changed, and now it should match a different service account
		update.deploymentReference.AddAll(r.findSubjectForBinding(obj)...)
	case *v1.ClusterRole:
		update.events = append(update.events, toRoleEvent(toRoxClusterRole(obj), action))
		if action == central.ResourceAction_REMOVE_RESOURCE {
			r.store.RemoveClusterRole(obj)
			update.events = append(update.events, r.mustGenerateRelatedEvents(obj, "", true)...)
		} else if action == central.ResourceAction_CREATE_RESOURCE || action == central.ResourceAction_SYNC_RESOURCE {
			r.store.UpsertClusterRole(obj)
			// In case the role is being created, or it's during sensor startup, dependent bindings should be processed.
			update.events = append(update.events, r.mustGenerateRelatedEvents(obj, string(obj.GetUID()), true)...)
		} else if action == central.ResourceAction_UPDATE_RESOURCE {
			r.store.UpsertClusterRole(obj)
		}
		update.deploymentReference.AddAll(r.findSubjectForRole(obj)...)
	case *v1.ClusterRoleBinding:
		update.deploymentReference.AddAll(r.findSubjectForBinding(obj)...)
		if action == central.ResourceAction_REMOVE_RESOURCE {
			r.store.RemoveClusterBinding(obj)
		} else {
			r.store.UpsertClusterBinding(obj)
		}
		roxBinding := r.toRoxClusterRoleBinding(obj)
		update.events = append(update.events, toBindingEvent(roxBinding, action))
		// Binding deployment reference has to be appended twice in case the binding changed, and now it should match a different service account
		update.deploymentReference.AddAll(r.findSubjectForBinding(obj)...)
	}
	return update
}

func (r *Dispatcher) findSubjectForBinding(binding metav1.Object) []namespacedSubject {
	return r.store.FindSubjectForBindingID(binding.GetNamespace(), binding.GetName(), string(binding.GetUID()))
}

func (r *Dispatcher) findSubjectForRole(role metav1.Object) []namespacedSubject {
	return r.store.FindSubjectForRole(role.GetNamespace(), role.GetName())
}

func (r *Dispatcher) mustGenerateRelatedEvents(obj metav1.Object, roleID string, isClusterRole bool) []*central.SensorEvent {
	// Only generate related binding events if re-sync is not enabled. Otherwise, binding events will be reprocessed every minute.
	relatedBindings := r.store.FindBindingForNamespacedRole(obj.GetNamespace(), obj.GetName())
	events, err := r.fetcher.generateManyDependentEvents(relatedBindings, roleID, isClusterRole)
	if err != nil {
		log.Warnf("failed to fetch related bindings: %s", err)
		return nil
	}
	return events
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
