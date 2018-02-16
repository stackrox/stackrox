package slack

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"text/template"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/notifications/notifiers"
	"bitbucket.org/stack-rox/apollo/pkg/notifications/types"
	"bitbucket.org/stack-rox/apollo/pkg/urlfmt"
)

var (
	log = logging.New("notifiers/slack")
)

const (
	colorCriticalAlert = "#FF2C4D"
	colorHighAlert     = "#FF634E"
	colorMediumAlert   = "#FF9365"
	colorLowAlert      = "#FFC780"
	colorDefault       = "warning"
)

// slack notifier plugin
type slack struct {
	config
	*v1.Notifier
}

// config for slack plugin
type config struct {
	Webhook string `json:"webhook"`
	Channel string `json:"channel"`
}

// notification json struct for richly-formatted notifications
type notification struct {
	Channel     string       `json:"channel" validate:"printascii"`
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

func (s *slack) getDescription(alert *v1.Alert) (string, error) {
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
func (s *slack) AlertNotify(alert *v1.Alert) error {
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
		Channel:     s.Channel,
	}
	jsonPayload, err := json.Marshal(&notification)
	if err != nil {
		return fmt.Errorf("Could not marshal notification for alert %v", alert.Id)
	}
	return postMessage(s.Webhook, jsonPayload)
}

// BenchmarkNotify takes in an benchmark schedule and generates the Slack message
func (s *slack) BenchmarkNotify(schedule *v1.BenchmarkSchedule) error {
	body, err := notifiers.FormatBenchmark(schedule, notifiers.BenchmarkLink(s.UiEndpoint))
	attachments := []attachment{
		{
			MarkDownFields: []string{"pretext", "text", "fields"},
			Text:           body,
		},
	}
	notification := notification{
		Attachments: attachments,
		Channel:     s.Channel,
	}
	jsonPayload, err := json.Marshal(&notification)
	if err != nil {
		return fmt.Errorf("Could not marshal notification for benchmark %v", schedule.GetName())
	}
	return postMessage(s.Webhook, jsonPayload)
}

func newSlack(protoNotifier *v1.Notifier) (*slack, error) {
	webhook, ok := protoNotifier.Config["webhook"]
	if !ok {
		return nil, fmt.Errorf("Webhook must be defined in the Slack Configuration")
	}
	channel, ok := protoNotifier.Config["channel"]
	if !ok {
		return nil, fmt.Errorf("Channel must be defined in the Slack Configuration")
	}

	webhook, err := urlfmt.FormatURL(webhook, true, false)
	if err != nil {
		return nil, err
	}

	return &slack{
		config: config{
			Webhook: webhook,
			Channel: channel,
		},
		Notifier: protoNotifier,
	}, nil
}

func (s *slack) ProtoNotifier() *v1.Notifier {
	return s.Notifier
}

func (s *slack) Test() error {
	n := notification{
		Channel: s.Channel,
		Text:    "This is a test message created to test integration with StackRox.",
	}
	jsonPayload, err := json.Marshal(&n)
	if err != nil {
		return errors.New("Could not marshal test notification")
	}
	return postMessage(s.Webhook, jsonPayload)
}

func postMessage(url string, jsonPayload []byte) (err error) {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil || resp == nil {
		log.Errorf("Error posting to slack: %v", err)
		return
	}
	defer resp.Body.Close()
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
func GetAttachmentColor(s v1.Severity) string {
	switch s {
	case v1.Severity_LOW_SEVERITY:
		return colorLowAlert
	case v1.Severity_MEDIUM_SEVERITY:
		return colorMediumAlert
	case v1.Severity_HIGH_SEVERITY:
		return colorHighAlert
	case v1.Severity_CRITICAL_SEVERITY:
		return colorCriticalAlert
	default:
		return colorDefault
	}
}

func init() {
	notifiers.Add("slack", func(notifier *v1.Notifier) (types.Notifier, error) {
		s, err := newSlack(notifier)
		return s, err
	})
}
