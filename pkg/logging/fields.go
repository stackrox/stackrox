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
	notifierField  = "notifier"
	errCodeField   = "err_code"
	alertIDField   = "alert_id"
)

var (
	resourceTypeFields = map[string]string{
		imageField:     resources.Image.String(),
		imageIDField:   resources.Image.String(),
		clusterIDField: resources.Cluster.String(),
		nodeIDField:    resources.Node.String(),
		notifierField:  "Notifier",
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

// NotifierName provides the notifier name as a structured log field.
func NotifierName(name string) zap.Field {
	return zap.String(notifierField, name)
}

// ErrCode refers to a specific human-readable error code. The error code
// will be used to identify a specific issue and may be used to render e.g. help information
// or can be used for filtering.
func ErrCode(code string) zap.Field {
	return zap.String(errCodeField, code)
}

// AlertID provides the alert ID as a structured log field.
func AlertID(id string) zap.Field {
	return zap.String(alertIDField, id)
}

// String provides a wrapper around zap.String and adds the key-value pair as structured log field.
// This should be _always_ preferred over direct calls to zap to minimize dependency to it.
func String(field, value string) zap.Field {
	return zap.String(field, value)
}

// Any provides a wrapper around zap.Any and adds the key-value pair as structured log field.
// This should be _always_ preferred over direct calls to zap to minimize dependency to it.
func Any(field string, value interface{}) zap.Field {
	return zap.Any(field, value)
}

// Strings provides a wrapper around zap.Strings and adds the key-value pair as structured log field.
// This should be _always_ preferred over direct calls to zap to minimize dependency to it.
func Strings(field string, values []string) zap.Field {
	return zap.Strings(field, values)
}

// Int provides a wrapper around zap.Int and adds the key-value pair as structured log field.
// This should be _always_ preferred over direct calls to zap to minimize dependency to it.
func Int(field string, value int) zap.Field {
	return zap.Int(field, value)
}

// getResourceTypeField returns whether the given zap.Field is related to a resource.
// If it is, it will return true and the name of the resource.
func getResourceTypeField(field zap.Field) (string, bool) {
	resource, exists := resourceTypeFields[field.Key]
	return resource, exists
}

func isIDField(fieldName string) bool {
	return fieldName != imageField && fieldName != notifierField
}
