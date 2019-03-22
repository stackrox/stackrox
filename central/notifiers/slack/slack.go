package slack

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/stackrox/rox/central/notifiers"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/urlfmt"
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
		return fmt.Errorf("Could not marshal notification for alert %v", alert.Id)
	}

	webhookURL := notifiers.GetLabelValue(alert, s.GetLabelKey(), s.GetLabelDefault())
	webhook, err := urlfmt.FormatURL(webhookURL, urlfmt.HTTPS, urlfmt.NoTrailingSlash)
	if err != nil {
		return err
	}

	return postMessage(webhook, jsonPayload)
}

// YamlNotify takes in a yaml file and generates the Slack message
func (s *slack) NetworkPolicyYAMLNotify(yaml string, clusterName string) error {
	if strings.Count(yaml, "\n") > 300 { // Looks like messages are truncated at ~340 lines.
		return fmt.Errorf("yaml is too large (>300 lines) to send over slack")
	}
	if len(yaml) > 35000 { // Slack hard limit is 40,000 characters, so leave 5,000 as a buffer to a round number.
		return fmt.Errorf("yaml is too large (>35,000 characters) to send over slack")
	}

	tagLine := fmt.Sprintf("*Network policy YAML to be applied on cluster '%s'*", clusterName)
	funcMap := template.FuncMap{
		"codeBlock": func(s string) string {
			return fmt.Sprintf("```\n%s\n```", s)
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
		return fmt.Errorf("Could not marshal notification for yaml for cluster %s", clusterName)
	}

	webhookURL := s.GetLabelDefault()
	webhook, err := urlfmt.FormatURL(webhookURL, urlfmt.HTTPS, urlfmt.NoTrailingSlash)
	if err != nil {
		return err
	}

	return postMessage(webhook, jsonPayload)

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

	return postMessage(webhook, jsonPayload)
}

func (s *slack) AckAlert(alert *storage.Alert) error {
	return nil
}

func (s *slack) ResolveAlert(alert *storage.Alert) error {
	return nil
}

func postMessage(url string, jsonPayload []byte) (err error) {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: timeout,
	}
	resp, err := client.Do(req)
	if err != nil || resp == nil {
		log.Errorf("Error posting to slack: %v", err)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		var bytes []byte
		bytes, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("Error reading slack response body: %v", err)
			return
		}
		log.Errorf("Slack error response: %v %v", resp.StatusCode, string(bytes))
	}
	return
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
