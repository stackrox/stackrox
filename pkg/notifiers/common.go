package notifiers

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events/option"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/retry"
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

var (
	log = logging.LoggerForModule(option.EnableAdministrationEvents())
)

// AlertLink is the link URL for this alert
func AlertLink(endpoint string, alert *storage.Alert) string {
	base, err := url.Parse(endpoint)
	if err != nil {
		log.Errorw("Invalid endpoint", logging.String("endpoint", endpoint), logging.Err(err))
		return ""
	}
	var alertPath string
	switch entity := alert.GetEntity().(type) {
	case *storage.Alert_Deployment_, *storage.Alert_Resource_:
		alertPath = fmt.Sprintf(alertLinkPath, alert.GetId())
	case *storage.Alert_Image:
		alertPath = fmt.Sprintf(imageLinkPath, entity.Image.GetId())
	}
	u, err := url.Parse(alertPath)
	if err != nil {
		log.Errorw("Invalid alert path found", logging.String("alert_path", alertPath), logging.Err(err))
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

// CreateError formats a returned HTTP response's status into an error, or nil.
func CreateError(notifier string, resp *http.Response, errCode string) error {
	if resp.StatusCode == 503 { // Error codes we want to retry go here.
		return retry.MakeRetryable(wrapError(notifier, resp, errCode))
	}
	return wrapError(notifier, resp, errCode)
}

func wrapError(notifier string, resp *http.Response, errCode string) error {
	if !httputil.Is2xxStatusCode(resp.StatusCode) {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrapf(err, "Error reading %s response body", notifier)
		}
		log.Errorw("Received an error response for notifier",
			logging.Err(err),
			logging.ErrCode(errCode),
			logging.NotifierName(notifier),
			logging.String("response", string(body)))
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
