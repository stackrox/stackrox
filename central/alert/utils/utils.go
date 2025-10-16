package utils

import "github.com/stackrox/rox/generated/storage"

// GetEntityType returns the type of entity for which the alert was raised
func GetEntityType(alert *storage.Alert) storage.Alert_EntityType {
	if alert == nil || alert.GetEntity() == nil {
		return storage.Alert_UNSET
	}
	switch alert.WhichEntity() {
	case storage.Alert_Deployment_case:
		return storage.Alert_DEPLOYMENT
	case storage.Alert_Image_case:
		return storage.Alert_CONTAINER_IMAGE
	case storage.Alert_Resource_case:
		return storage.Alert_RESOURCE
	}
	return storage.Alert_UNSET
}
