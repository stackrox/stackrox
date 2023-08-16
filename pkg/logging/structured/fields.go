package structured

import (
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
		imageField:     "Image",
		imageIDField:   "Image",
		clusterIDField: "Cluster",
		nodeIDField:    "Node",
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

// IsResourceTypeField returns whether the given zap.Field is related to a resource.
// If it is, it will return true and the name of the resource.
func IsResourceTypeField(field zap.Field) (bool, string) {
	resource, exists := resourceTypeFields[field.Key]
	return exists, resource
}
