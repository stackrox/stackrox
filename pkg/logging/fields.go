package logging

import (
	"context"

	administrationResources "github.com/stackrox/rox/pkg/administration/events/resources"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
)

const (
	ClusterIDContextValue = "stackrox.cluster-id"

	alertIDField      = "alert_id"
	apiTokenIDField   = "api_token_id"
	apiTokenNameField = "api_token_name"
	backupField       = "backup"
	cloudSourceField  = "cloud_source"
	clusterIDField    = "cluster_id"
	clusterNameField  = "cluster_name"
	errCodeField      = "err_code"
	imageField        = "image"
	imageIDField      = "image_id"
	nodeIDField       = "node_id"
	notifierField     = "notifier"
)

var resourceTypeFields = map[string]string{
	apiTokenIDField:   administrationResources.APIToken,
	apiTokenNameField: administrationResources.APIToken,
	backupField:       administrationResources.Backup,
	cloudSourceField:  administrationResources.CloudSource,
	clusterIDField:    administrationResources.Cluster,
	imageField:        administrationResources.Image,
	imageIDField:      administrationResources.Image,
	nodeIDField:       administrationResources.Node,
	notifierField:     administrationResources.Notifier,
}

// Err wraps err into a zap.Field instance with a well-known name 'error'.
func Err(err error) zap.Field {
	return zap.Error(err)
}

// Context provides selected context values as structured log fields.
func Context(ctx context.Context) zap.Field {
	// Use zap.Dict in zap v1.26.0.
	// return zap.Dict("context",
	// 	zap.NamedError("cause", context.Cause(ctx)),
	// 	zap.Any("clusterID", ctx.Value("clusterID")),
	// )

	contextMap := make(map[string]interface{})
	if cause := context.Cause(ctx); cause != nil {
		contextMap["cause"] = cause.Error()
	}
	if clusterID := metadata.ValueFromIncomingContext(ctx, ClusterIDContextValue); clusterID != nil {
		contextMap["clusterID"] = clusterID
	}
	if len(contextMap) > 0 {
		return zap.Any("context", contextMap)
	}
	return zap.Skip()
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

// BackupName provides the backup name as a structured log field.
func BackupName(name string) zap.Field {
	return zap.String(backupField, name)
}

// CloudSourceName provides the cloud source name as a structured log field.
func CloudSourceName(name string) zap.Field {
	return zap.String(cloudSourceField, name)
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

// APITokenID provides the API token ID as a structured log field.
func APITokenID(id string) zap.Field {
	return zap.String(apiTokenIDField, id)
}

// APITokenName provides the API token name as a structured log field.
func APITokenName(name string) zap.Field {
	return zap.String(apiTokenNameField, name)
}

// ClusterName provides the cluster name as a structured log field.
func ClusterName(name string) zap.Field {
	return zap.String(clusterNameField, name)
}

// Wrapper functions for zap.Field functions.

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

// Bool provides a wrapper around zap.Bool and adds the key-value pair as structured log field.
// This should be _always_ preferred over direct calls to zap to minimize dependency to it.
func Bool(field string, value bool) zap.Field {
	return zap.Bool(field, value)
}

// End Wrapper functions for zap.field functions.

// Helper functions.

// getResourceTypeField returns whether the given zap.Field is related to a resource.
// If it is, it will return true and the name of the resource.
func getResourceTypeField(field zap.Field) (string, bool) {
	resource, exists := resourceTypeFields[field.Key]
	return resource, exists
}

func isIDField(fieldName string) bool {
	return fieldName != apiTokenNameField &&
		fieldName != backupField &&
		fieldName != cloudSourceField &&
		fieldName != imageField &&
		fieldName != notifierField
}
