package fixtures

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/uuid"
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
