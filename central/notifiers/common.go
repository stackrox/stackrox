package notifiers

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/retry"
)

const (
	alertLinkPath = "/main/violations/%s"
)

// AlertLink is the link URL for this alert
func AlertLink(endpoint string, alertID string) string {
	base, err := url.Parse(endpoint)
	if err != nil {
		log.Error(err)
	}
	alertPath := fmt.Sprintf(alertLinkPath, alertID)
	u, err := url.Parse(alertPath)
	if err != nil {
		log.Error(err)
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

// CreateError formats a returned HTTP response's status into an error, or nil.
func CreateError(notifier string, resp *http.Response) error {
	if resp.StatusCode == 503 { // Error codes we want to retry go here.
		return retry.MakeRetryable(wrapError(notifier, resp))
	}
	return wrapError(notifier, resp)
}

func wrapError(notifier string, resp *http.Response) error {
	if !httputil.Is2xxStatusCode(resp.StatusCode) {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrapf(err, "Error reading %s response body", notifier)
		}
		log.Errorf("Received error response from %s: %d %s", notifier, resp.StatusCode, string(body))
		return errors.Errorf("Received error response from %s: %d. Check central logs for full error.", notifier, resp.StatusCode)
	}
	return nil
}
