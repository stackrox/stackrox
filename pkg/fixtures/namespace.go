package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
)

// GetNamespace returns a mock `*storage.NamespaceMetadata` object.
func GetNamespace(clusterID, clusterName, namespace string) *storage.NamespaceMetadata {
	return &storage.NamespaceMetadata{
		Id:          uuid.NewV4().String(),
		Name:        namespace,
		ClusterId:   clusterID,
		ClusterName: clusterName,
	}
}

// GetScopedNamespace returns a mock *storage.NamespaceMetadata object.
func GetScopedNamespace(ID string, clusterID string, namespace string) *storage.NamespaceMetadata {
	return &storage.NamespaceMetadata{
		Id:          ID,
		Name:        namespace,
		ClusterId:   clusterID,
		ClusterName: clusterID,
	}
}
