package logging

import (
	"github.com/stackrox/rox/pkg/sac/resources"
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
