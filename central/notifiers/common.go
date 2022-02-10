package notifiers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/search"
)

const (
	colorCriticalAlert = "#FF2C4D"
	colorHighAlert     = "#FF634E"
	colorMediumAlert   = "#FF9365"
	colorLowAlert      = "#FFC780"
	colorDefault       = "warning"

	// YAMLNotificationColor is color of YAML notification used by slack, teams etc.
	YAMLNotificationColor = "#FF9365"
	// Timeout is timeout for HTTP requests sent to various integrations such as slack, teams etc.
	Timeout = 10 * time.Second
)

const (
	alertLinkPath = "/main/violations/%s"
	imageLinkPath = "/main/vulnerability-management/image/%s"
)

// AlertLink is the link URL for this alert
func AlertLink(endpoint string, alert *storage.Alert) string {
	base, err := url.Parse(endpoint)
	if err != nil {
		log.Errorf("Invalid endpoint %s: %v", endpoint, err)
		return ""
	}
	var alertPath string
	switch entity := alert.GetEntity().(type) {
	case *storage.Alert_Deployment_:
		alertPath = fmt.Sprintf(alertLinkPath, alert.GetId())
	case *storage.Alert_Image:
		alertPath = fmt.Sprintf(imageLinkPath, entity.Image.GetId())
	}
	u, err := url.Parse(alertPath)
	if err != nil {
		log.Errorf("Invalid alert path %s: %v", alertPath, err)
		return ""
	}
	return base.ResolveReference(u).String()
}

// SeverityString is the nice form of the Severity enum
func SeverityString(s storage.Severity) string {
	switch s {
	case storage.Severity_UNSET_SEVERITY:
		return "Unset"
	case storage.Severity_LOW_SEVERITY:
		return "Low"
	case storage.Severity_MEDIUM_SEVERITY:
		return "Medium"
	case storage.Severity_HIGH_SEVERITY:
		return "High"
	case storage.Severity_CRITICAL_SEVERITY:
		return "Critical"
	default:
		panic("The severity enum has been updated, but this switch statement hasn't")
	}
}

// GetAnnotationValue returns the value of the annotation with the key annotationKey on the deployment or namespace of the alert.
// It will attempt to get it from the deployment, but if it doesn't exist it will get it from the namespace. If neither exists, it will return the default value.
// This value from the annotation is used by certain notifiers to redirect notifications to other channels. For example, the email notifier can send to an alternate email depending on the annotation value.
// NOTE: It is possible that this will pull the value from a deployment label instead of annotation. This remains for backwards compatibility purposes, because versions <63.0 supported this on labels and annotations.
func GetAnnotationValue(ctx context.Context, alert *storage.Alert, annotationKey, defaultValue string, namespaceStore namespaceDataStore.DataStore) string {
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
	if ns := getNamespaceFromAlert(ctx, alert, namespaceStore); ns != nil {
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

	if namespaceName == "" || clusterID == "" {
		log.Errorf("Alert entity doesn't contain namespace and cluster ID: %+v", alert.GetEntity())
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

// CreateError formats a returned HTTP response's status into an error, or nil.
func CreateError(notifier string, resp *http.Response) error {
	if resp.StatusCode == 503 { // Error codes we want to retry go here.
		return retry.MakeRetryable(wrapError(notifier, resp))
	}
	return wrapError(notifier, resp)
}

func wrapError(notifier string, resp *http.Response) error {
	if !httputil.Is2xxStatusCode(resp.StatusCode) {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrapf(err, "Error reading %s response body", notifier)
		}
		log.Errorf("Received error response from %s: %d %s", notifier, resp.StatusCode, string(body))
		return errors.Errorf("Received error response from %s: %d. Check central logs for full error.", notifier, resp.StatusCode)
	}
	return nil
}

// GetAttachmentColor returns the corresponding color for each severity.
func GetAttachmentColor(s storage.Severity) string {
	switch s {
	case storage.Severity_LOW_SEVERITY:
		return colorLowAlert
	case storage.Severity_MEDIUM_SEVERITY:
		return colorMediumAlert
	case storage.Severity_HIGH_SEVERITY:
		return colorHighAlert
	case storage.Severity_CRITICAL_SEVERITY:
		return colorCriticalAlert
	default:
		return colorDefault
	}
}
