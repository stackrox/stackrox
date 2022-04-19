package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
)

// GetCluster returns a mock `*storage.Cluster` object.
func GetCluster(name string, labels map[string]string) *storage.Cluster {
	return &storage.Cluster{
		Id:     uuid.NewV4().String(),
		Name:   name,
		Labels: labels,
	}
}
