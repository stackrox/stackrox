package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/notifiers"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	log = logging.LoggerForModule()
)

const (
	colorCriticalAlert = "#FF2C4D"
	colorHighAlert     = "#FF634E"
	colorMediumAlert   = "#FF9365"
	colorLowAlert      = "#FFC780"
	colorDefault       = "warning"

	timeout = 10 * time.Second
)

// slack notifier plugin
type slack struct {
	*storage.Notifier
}

// notification json struct for richly-formatted notifications
type notification struct {
	Attachments []attachment `json:"attachments"`
	Text        string       `json:"text"`
}

// Attachment json struct for attachments
type attachment struct {
	FallBack       string            `json:"fallback"`
	Color          string            `json:"color"`
	Pretext        string            `json:"pretext"`
	Title          string            `json:"title"`
	Text           string            `json:"text"`
	MarkDownFields []string          `json:"mrkdwn_in"`
	Fields         []attachmentField `json:"fields"`
}

// attachmentField json struct for attachment fields
type attachmentField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

func (s *slack) getDescription(alert *storage.Alert) (string, error) {
	tabSpace := "        "
	dblTabSpace := tabSpace + tabSpace
	funcMap := template.FuncMap{
		"header": func(s string) string {
			return fmt.Sprintf("\r\n*%v*\r\n", s)
		},
		"subheader": func(s string) string {
			return fmt.Sprintf("\r\n%v*%v*\r\n", tabSpace, s)
		},
		"line": func(s string) string {
			return fmt.Sprintf("%v\r\n", s)
		},
		"list": func(s string) string {
			return fmt.Sprintf("%v    - %v\r\n", tabSpace, s)
		},
		"nestedList": func(s string) string {
			return fmt.Sprintf("%v- %v\r\n", dblTabSpace, s)
		},
	}
	alertLink := notifiers.AlertLink(s.Notifier.UiEndpoint, alert.GetId())
	return notifiers.FormatPolicy(alert, alertLink, funcMap)
}

// AlertNotify takes in an alert and generates the Slack message
func (s *slack) AlertNotify(alert *storage.Alert) error {
	tagLine := fmt.Sprintf("*Deployment %v (%v) violates '%v' Policy*", alert.Deployment.Name, alert.Deployment.Id, alert.Policy.Name)
	body, err := s.getDescription(alert)
	if err != nil {
		return err
	}
	attachments := []attachment{
		{
			FallBack:       body,
			Color:          GetAttachmentColor(alert.GetPolicy().GetSeverity()),
			Pretext:        tagLine,
			Text:           body,
			MarkDownFields: []string{"pretext", "text", "fields"},
		},
	}
	notification := notification{
		Attachments: attachments,
	}
	jsonPayload, err := json.Marshal(&notification)
	if err != nil {
		return errors.Errorf("Could not marshal notification for alert %v", alert.Id)
	}

	webhookURL := notifiers.GetLabelValue(alert, s.GetLabelKey(), s.GetLabelDefault())
	webhook, err := urlfmt.FormatURL(webhookURL, urlfmt.HTTPS, urlfmt.NoTrailingSlash)
	if err != nil {
		return err
	}

	return retry.WithRetry(
		func() error {
			return postMessage(webhook, jsonPayload)
		},
		retry.OnlyRetryableErrors(),
		retry.Tries(3),
		retry.BetweenAttempts(func(previousAttempt int) {
			wait := time.Duration(previousAttempt * previousAttempt * 100)
			time.Sleep(wait * time.Millisecond)
		}),
	)
}

// YamlNotify takes in a yaml file and generates the Slack message
func (s *slack) NetworkPolicyYAMLNotify(yaml string, clusterName string) error {
	if strings.Count(yaml, "\n") > 300 { // Looks like messages are truncated at ~340 lines.
		return errors.Errorf("yaml is too large (>300 lines) to send over slack")
	}
	if len(yaml) > 35000 { // Slack hard limit is 40,000 characters, so leave 5,000 as a buffer to a round number.
		return errors.Errorf("yaml is too large (>35,000 characters) to send over slack")
	}

	tagLine := fmt.Sprintf("*Network policy YAML to be applied on cluster '%s'*", clusterName)
	funcMap := template.FuncMap{
		"codeBlock": func(s string) string {
			if len(s) > 0 {
				return fmt.Sprintf("```\n%s\n```", s)
			}
			return "```\n<YAML is empty>\n```"
		},
	}
	body, err := notifiers.FormatNetworkPolicyYAML(yaml, clusterName, funcMap)
	if err != nil {
		return err
	}
	attachments := []attachment{
		{
			FallBack:       body,
			Color:          colorMediumAlert,
			Pretext:        tagLine,
			Text:           body,
			MarkDownFields: []string{"pretext", "text", "fields"},
		},
	}
	notification := notification{
		Attachments: attachments,
	}
	jsonPayload, err := json.Marshal(&notification)
	if err != nil {
		return errors.Errorf("Could not marshal notification for yaml for cluster %s", clusterName)
	}

	webhookURL := s.GetLabelDefault()
	webhook, err := urlfmt.FormatURL(webhookURL, urlfmt.HTTPS, urlfmt.NoTrailingSlash)
	if err != nil {
		return err
	}

	return retry.WithRetry(
		func() error {
			return postMessage(webhook, jsonPayload)
		},
		retry.OnlyRetryableErrors(),
		retry.Tries(3),
		retry.BetweenAttempts(func(previousAttempt int) {
			wait := time.Duration(previousAttempt * previousAttempt * 100)
			time.Sleep(wait * time.Millisecond)
		}),
	)
}

func newSlack(notifier *storage.Notifier) (*slack, error) {
	return &slack{
		Notifier: notifier,
	}, nil
}

func (s *slack) ProtoNotifier() *storage.Notifier {
	return s.Notifier
}

func (s *slack) Test() error {
	n := notification{
		Text: "This is a test message created to test integration with StackRox.",
	}
	jsonPayload, err := json.Marshal(&n)
	if err != nil {
		return errors.New("Could not marshal test notification")
	}

	webhook, err := urlfmt.FormatURL(s.GetLabelDefault(), urlfmt.HTTPS, urlfmt.NoTrailingSlash)
	if err != nil {
		return err
	}

	return retry.WithRetry(
		func() error {
			return postMessage(webhook, jsonPayload)
		},
		retry.OnlyRetryableErrors(),
		retry.Tries(3),
		retry.BetweenAttempts(func(previousAttempt int) {
			wait := time.Duration(previousAttempt * previousAttempt * 100)
			time.Sleep(wait * time.Millisecond)
		}),
	)
}

func postMessage(url string, jsonPayload []byte) error {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		return err
	}

	client := &http.Client{
		Timeout: timeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Error posting to slack: %v", err)
		return errors.Wrap(err, "Error posting to slack")
	}
	defer utils.IgnoreError(resp.Body.Close)

	return notifiers.CreateError("Slack", resp)
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

func init() {
	notifiers.Add("slack", func(notifier *storage.Notifier) (notifiers.Notifier, error) {
		s, err := newSlack(notifier)
		return s, err
	})
}
