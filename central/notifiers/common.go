package notifiers

import (
	"fmt"
	"log"
	"net/url"

	"github.com/stackrox/rox/generated/storage"
)

const (
	alertLinkPath     = "/main/violations/%s"
	benchmarkLinkPath = "/main/compliance"
)

// AlertLink is the link URL for this alert
func AlertLink(endpoint string, alertID string) string {
	base, err := url.Parse(endpoint)
	if err != nil {
		log.Print(err)
	}
	alertPath := fmt.Sprintf(alertLinkPath, alertID)
	u, err := url.Parse(alertPath)
	if err != nil {
		log.Print(err)
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

// GetLabelValue returns the value based on the label in the deployment or the default value if it does not exist
func GetLabelValue(alert *storage.Alert, labelKey, def string) string {
	deployment := alert.GetDeployment()
	// Annotations will most likely be used for k8s
	if value, ok := deployment.GetAnnotations()[labelKey]; ok {
		return value
	}
	// Labels will most likely be used for docker
	if value, ok := deployment.GetLabels()[labelKey]; ok {
		return value
	}
	return def
}
