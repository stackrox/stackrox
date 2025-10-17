package utils

import "github.com/stackrox/rox/generated/storage"

// GetEntityType returns the type of entity for which the alert was raised
func GetEntityType(alert *storage.Alert) storage.Alert_EntityType {
	if alert == nil || alert.GetEntity() == nil {
		return storage.Alert_UNSET
	}
	switch alert.GetEntity().(type) {
	case *storage.Alert_Deployment_:
		return storage.Alert_DEPLOYMENT
	case *storage.Alert_Image:
		return storage.Alert_CONTAINER_IMAGE
	case *storage.Alert_Resource_:
		return storage.Alert_RESOURCE
	case *storage.Alert_Node_:
		return storage.Alert_NODE
	}
	return storage.Alert_UNSET
}
