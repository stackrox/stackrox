package notifiers

import (
	"fmt"
	"log"
	"net/url"

	"github.com/stackrox/rox/generated/api/v1"
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

// BenchmarkLink is the link URL for this alert
func BenchmarkLink(endpoint string) string {
	base, err := url.Parse(endpoint)
	if err != nil {
		log.Print(err)
	}
	u, err := url.Parse(benchmarkLinkPath)
	if err != nil {
		log.Print(err)
		return ""
	}
	return base.ResolveReference(u).String()
}

// SeverityString is the nice form of the Severity enum
func SeverityString(s v1.Severity) string {
	switch s {
	case v1.Severity_UNSET_SEVERITY:
		return "Unset"
	case v1.Severity_LOW_SEVERITY:
		return "Low"
	case v1.Severity_MEDIUM_SEVERITY:
		return "Medium"
	case v1.Severity_HIGH_SEVERITY:
		return "High"
	case v1.Severity_CRITICAL_SEVERITY:
		return "Critical"
	default:
		panic("The severity enum has been updated, but this switch statement hasn't")
	}
}

// GetLabelValue returns the value based on the label in the deployment or the default value if it does not exist
func GetLabelValue(alert *v1.Alert, labelKey, def string) string {
	deployment := alert.GetDeployment()
	// Annotations will most likely be used for k8s
	for _, annotation := range deployment.GetAnnotations() {
		if annotation.GetKey() == labelKey {
			return annotation.GetValue()
		}
	}
	// Labels will most likely be used for docker
	for _, label := range deployment.GetLabels() {
		if label.GetKey() == labelKey {
			return label.GetValue()
		}
	}
	return def
}
