package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
)

// GetNamespace returns a mock `*storage.NamespaceMetadata` object.
func GetNamespace(clusterID, clusterName, namespace string) *storage.NamespaceMetadata {
	nm := &storage.NamespaceMetadata{}
	nm.SetId(uuid.NewV4().String())
	nm.SetName(namespace)
	nm.SetClusterId(clusterID)
	nm.SetClusterName(clusterName)
	return nm
}

// GetScopedNamespace returns a mock *storage.NamespaceMetadata object.
func GetScopedNamespace(ID string, clusterID string, namespace string) *storage.NamespaceMetadata {
	nm := &storage.NamespaceMetadata{}
	nm.SetId(ID)
	nm.SetName(namespace)
	nm.SetClusterId(clusterID)
	nm.SetClusterName(clusterID)
	return nm
}
