package rbac

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	v1 "k8s.io/api/rbac/v1"
)

// NamespacedServiceAccount keeps a pair of service account and used namespace.
type NamespacedServiceAccount interface {
	GetServiceAccount() string
	GetNamespace() string
}

// Store handles correlating updates to K8s rbac types and generates events from them.
type Store interface {
	UpsertRole(role *v1.Role) *storage.K8SRole
	RemoveRole(role *v1.Role) *storage.K8SRole

	UpsertClusterRole(role *v1.ClusterRole) *storage.K8SRole
	RemoveClusterRole(role *v1.ClusterRole) *storage.K8SRole

	UpsertBinding(binding *v1.RoleBinding) *storage.K8SRoleBinding
	RemoveBinding(binding *v1.RoleBinding) *storage.K8SRoleBinding

	UpsertClusterBinding(binding *v1.ClusterRoleBinding) *storage.K8SRoleBinding
	RemoveClusterBinding(binding *v1.ClusterRoleBinding) *storage.K8SRoleBinding
	GetPermissionLevelForDeployment(deployment NamespacedServiceAccount) storage.PermissionLevel
}

// NewStore creates a new instance of Store
func NewStore(syncedFlag *concurrency.Flag) Store {
	return &storeImpl{
		syncedFlag: syncedFlag,
		roles:      make(map[namespacedRoleRef]*storage.K8SRole),

		bindingsByID:       make(map[string]*storage.K8SRoleBinding),
		bindingIDToRoleRef: make(map[string]namespacedRoleRef),
		roleRefToBindings:  make(map[namespacedRoleRef]map[string]*storage.K8SRoleBinding),

		// Incredibly unlikely that there are no roles and no bindings, but for safety initialize empty buckets
		bucketEvaluator: newBucketEvaluator(nil, nil),
	}
}
