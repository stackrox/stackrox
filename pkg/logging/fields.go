package logging

import (
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/set"
	"go.uber.org/zap"
)

const (
	imageField     = "image"
	clusterIDField = "cluster_id"
	imageIDField   = "image_id"
	nodeIDField    = "node_id"
)

var (
	resourceTypeFields = map[string]string{
		imageField:     resources.Image.String(),
		imageIDField:   resources.Image.String(),
		clusterIDField: resources.Cluster.String(),
		nodeIDField:    resources.Node.String(),
	}

	// Defines the priority between resource type fields. In case multiple resource types are specified, the ones with
	// the higher priority will be take higher priority and used for things like the resource type of the event.
	resourceTypePriority = []set.StringSet{
		set.NewStringSet(imageIDField, clusterIDField, imageField),
		set.NewStringSet(clusterIDField),
	}
)

// Err wraps err into a zap.Field instance with a well-known name 'error'.
func Err(err error) zap.Field {
	return zap.Error(err)
}

// ImageName provides the image name as a structured log field.
func ImageName(name string) zap.Field {
	return zap.String(imageField, name)
}

// ClusterID provides the cluster ID as a structured log field.
func ClusterID(id string) zap.Field {
	return zap.String(clusterIDField, id)
}

// ImageID provides the image ID as a structured log field.
func ImageID(id string) zap.Field {
	return zap.String(imageIDField, id)
}

// NodeID provides the node ID as a structured log field.
func NodeID(id string) zap.Field {
	return zap.String(nodeIDField, id)
}

// getResourceTypeField returns whether the given zap.Field is related to a resource.
// If it is, it will return true and the name of the resource.
func getResourceTypeField(field zap.Field) (string, bool) {
	resource, exists := resourceTypeFields[field.Key]
	return resource, exists
}

// getHigherPriorityResourceField returns true if the new resource is higher priority than the previous resource.
// Note that when both fields are equal in priority, the existing resource will be kept and false will be returned.
func getHigherPriorityResourceField(newResource string, existingResource string) bool {
	for priority := range resourceTypePriority {
		if resourceTypePriority[priority].Contains(existingResource) {
			return false
		}
		if resourceTypePriority[priority].Contains(newResource) {
			return true
		}
	}
	return false
}
