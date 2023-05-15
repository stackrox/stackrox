package metadatagetter

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// NamespaceAnnotationStore represents the functions needed to resolve notification metadata.
type NamespaceAnnotationStore interface {
	GetAnnotationsForNamespace(name string) map[string]string
}

type memStoreMetadataGetter struct {
	nsStore NamespaceAnnotationStore
}

// GetAnnotationValue returns the value of the annotation with the key annotationKey on the deployment or namespace of the alert.
// It will attempt to get it from the deployment, but if it doesn't exist it will get it from the namespace. If neither exists, it will return the default value.
// This value from the annotation is used by certain notifiers to redirect notifications to other channels. For example, the email notifier can send to an alternate email depending on the annotation value.
// NOTE: It is possible that this will pull the value from a deployment label instead of annotation. This remains for backwards compatibility purposes, because versions <63.0 supported this on labels and annotations.
func (m memStoreMetadataGetter) GetAnnotationValue(_ context.Context, alert *storage.Alert, annotationKey, defaultValue string) string {
	// Skip entire processing if the label key is not even set
	if annotationKey == "" {
		return defaultValue
	}

	// Try get annotation from deployment
	if deployment := alert.GetDeployment(); deployment != nil {
		if value, ok := deployment.GetAnnotations()[annotationKey]; ok {
			return value
		}

		// Note: Label support was added for Docker Swarm and most notifiers won't even work with labels
		// because labels cannot have emails or URLs. But it is theoretically possible to store a JIRA project in there
		// and for backwards compatibility with the users that are using labels, we will continue to read
		if value, ok := deployment.GetLabels()[annotationKey]; ok {
			return value
		}
	}

	// Otherwise get annotation from namespace
	if ns := m.getNamespaceFromAlert(alert); ns != "" {
		annotationsForNs := m.nsStore.GetAnnotationsForNamespace(ns)
		if annotationsForNs == nil {
			return defaultValue
		}
		if value, ok := annotationsForNs[annotationKey]; ok {
			return value
		}
	}

	// If neither exists, fallback
	return defaultValue
}

// Tries to fetch the NamespaceMetadata object given the namespace name within the alert.
func (m memStoreMetadataGetter) getNamespaceFromAlert(alert *storage.Alert) string {
	var namespaceName string
	switch entity := alert.GetEntity().(type) {
	case *storage.Alert_Deployment_:
		namespaceName = entity.Deployment.GetNamespace()
	case *storage.Alert_Resource_:
		namespaceName = entity.Resource.GetNamespace()
	// we really can't have image alerts here in the admission controller
	case *storage.Alert_Image:
		// An image doesn't have a namespace, but it's not an error so just return
		return ""
	default:
		log.Error("Unexpected entity in alert")
		return ""
	}
	if namespaceName == "" {
		log.Errorf("Alert entity doesn't contain namespace name: %+v", alert.GetEntity())
		return ""
	}
	return namespaceName
}

func newMetadataGetter(nsStore NamespaceAnnotationStore) *memStoreMetadataGetter {
	return &memStoreMetadataGetter{
		nsStore: nsStore,
	}
}
