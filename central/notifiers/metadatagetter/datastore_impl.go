package metadatagetter

import (
	"context"
	"testing"

	namespaceDataStore "github.com/stackrox/rox/central/namespace/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/search"
)

var (
	log = logging.LoggerForModule()
)

type datastoreMetadataGetter struct {
	datastore namespaceDataStore.DataStore
}

func newMetadataGetter() *datastoreMetadataGetter {
	return &datastoreMetadataGetter{
		datastore: namespaceDataStore.Singleton(),
	}
}

// newTestMetadataGetter returns an instance of notifiers.MetadataGetter for testing purposes
func newTestMetadataGetter(t *testing.T, store namespaceDataStore.DataStore) notifiers.MetadataGetter {
	if t == nil {
		return nil
	}
	return &datastoreMetadataGetter{
		datastore: store,
	}
}

// GetAnnotationValue returns the value of the annotation with the key annotationKey on the deployment or namespace of the alert.
// It will attempt to get it from the deployment, but if it doesn't exist it will get it from the namespace. If neither exists, it will return the default value.
// This value from the annotation is used by certain notifiers to redirect notifications to other channels. For example, the email notifier can send to an alternate email depending on the annotation value.
// NOTE: It is possible that this will pull the value from a deployment label instead of annotation. This remains for backwards compatibility purposes, because versions <63.0 supported this on labels and annotations.
func (m datastoreMetadataGetter) GetAnnotationValue(ctx context.Context, alert *storage.Alert, annotationKey, defaultValue string) string {
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
	if ns := getNamespaceFromAlert(ctx, alert, m.datastore); ns != nil {
		if value, ok := ns.GetAnnotations()[annotationKey]; ok {
			return value
		}
	}

	// If neither exists, fallback
	return defaultValue
}

// Tries to fetch the NamespaceMetadata object given the namespace name within the alert.
func getNamespaceFromAlert(ctx context.Context, alert *storage.Alert, namespaceStore namespaceDataStore.DataStore) *storage.NamespaceMetadata {
	var namespaceName, clusterID string
	switch entity := alert.GetEntity().(type) {
	case *storage.Alert_Deployment_:
		namespaceName = entity.Deployment.GetNamespace()
		clusterID = entity.Deployment.GetClusterId()
	case *storage.Alert_Resource_:
		namespaceName = entity.Resource.GetNamespace()
		clusterID = entity.Resource.GetClusterId()
	case *storage.Alert_Image:
		// An image doesn't have a namespace, but it's not an error so just return
		return nil
	default:
		log.Error("Unexpected entity in alert")
		return nil
	}

	if namespaceName == "" {
		return nil
	}

	if clusterID == "" {
		log.Errorf("Alert entity doesn't contain cluster ID: %+v", alert.GetEntity())
		return nil
	}

	q := search.NewQueryBuilder().AddExactMatches(search.Namespace, namespaceName).AddExactMatches(search.ClusterID, clusterID).ProtoQuery()
	namespaces, err := namespaceStore.SearchNamespaces(ctx, q)

	if err != nil {
		log.Errorf("Failed to find namespace %s in cluster %s from alert with error %v", namespaceName, clusterID, err)
		return nil
	}

	if len(namespaces) != 1 {
		log.Errorf("Failed to find the specific namespace %s in cluster %s from alert; instead found: %+v", namespaceName, clusterID, namespaces)
		return nil
	}

	return namespaces[0]
}

func (m datastoreMetadataGetter) GetNamespaceLabels(ctx context.Context, alert *storage.Alert) map[string]string {
	if ns := getNamespaceFromAlert(ctx, alert, m.datastore); ns != nil {
		if labels := ns.GetLabels(); labels != nil {
			return labels
		}
	}
	return map[string]string{}
}
