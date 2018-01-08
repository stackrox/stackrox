package slack

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/images"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/notifications/notifiers"
	"bitbucket.org/stack-rox/apollo/pkg/notifications/types"
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

func codeBlock(str string) string {
	return "```" + str + "```"
}

func inline(str string) string {
	return "`" + str + "`"
}

// Notify takes in an alert and the portal endpoint and generates the Slack message
func (s *slack) Notify(alert *v1.Alert) error {
	tagLine := fmt.Sprintf("Deployment %v (%v) violates '%v' Policy", alert.Deployment.Name, alert.Deployment.Id, alert.Policy.Name)
	endpoint := notifiers.AlertLink(alert, s.UiEndpoint)
	pretext := fmt.Sprintf("<%v|%v>", endpoint, tagLine)

	policy := alert.GetPolicy()

	attachmentFields := []attachmentField{
		{
			Title: "",
			Value: "*Severity*: " + inline(notifiers.SeverityString(policy.GetSeverity())),
		},
		{
			Title: "Policy Description",
			Value: codeBlock(policy.GetDescription()),
		},
		{
			Title: "Violations",
			Value: fmt.Sprintf("```%s```", strings.Join(notifiers.StringViolations(alert.GetViolations()), "\n")),
		},
		{
			Title: "Deployment",
			Value: fmt.Sprintf("```Name : %v\nImage: %v```",
				alert.Deployment.Name, images.FromContainers(alert.GetDeployment().GetContainers()).String()),
		},
	}
	attachments := []attachment{
		{
			FallBack:       "Rox Alert",
			Color:          getAttachmentColor(policy.GetSeverity()),
			Pretext:        pretext,
			Text:           "",
			MarkDownFields: []string{"text", "fields"},
			Fields:         attachmentFields,
		},
	}
	notification := notification{
		Attachments: attachments,
		Channel:     s.Channel,
	}
	jsonPayload, err := json.Marshal(&notification)
	log.Debugf("JSON Payload: % #+v", string(jsonPayload))
	if err != nil {
		return fmt.Errorf("Could not marshal notification for alert %v", alert.Id)
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
		Attachments: []attachment{
			{
				FallBack:       "Rox Alert",
				Pretext:        "This is a test alert from StackRox",
				Text:           "",
				MarkDownFields: []string{"text", "fields"},
			},
		},
	}
	jsonPayload, err := json.Marshal(&n)
	log.Debugf("JSON Payload: % #+v", string(jsonPayload))
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

func getAttachmentColor(s v1.Severity) string {
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
