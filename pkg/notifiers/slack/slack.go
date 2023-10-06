package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events/codes"
	"github.com/stackrox/rox/pkg/administration/events/option"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/logging"
	mitreDS "github.com/stackrox/rox/pkg/mitre/datastore"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	tabSpace      = "        "
	dblTabSpace   = tabSpace + tabSpace
	threeTabSpace = dblTabSpace + tabSpace
)

var (
	log = logging.LoggerForModule(option.EnableAdministrationEvents())
)

// slack notifier plugin
type slack struct {
	*storage.Notifier
	client *http.Client

	metadataGetter notifiers.MetadataGetter
	mitreStore     mitreDS.AttackReadOnlyDataStore
}

// notification json struct for richly-formatted notifications
type notification struct {
	Attachments []attachment `json:"attachments"`
	Text        string       `json:"text"`
}

// attachment json struct for attachments
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
		"section": func(s string) string {
			return fmt.Sprintf("\r\n%v*%v*\r\n", dblTabSpace, s)
		},
		"group": func(s string) string {
			return fmt.Sprintf("\r\n%v*%v*", threeTabSpace, s)
		},
		"valuePrinter": func(values []*storage.PolicyValue, op storage.BooleanOperator, negated bool) string {
			var opString string
			if op == storage.BooleanOperator_OR {
				opString = " OR "
			} else {
				opString = " AND "
			}

			var valueStrings []string
			for _, value := range values {
				codeString := fmt.Sprintf("`%s`", value.GetValue())
				valueStrings = append(valueStrings, codeString)
			}

			valuesString := strings.Join(valueStrings, opString)
			if negated {
				valuesString = fmt.Sprintf("NOT (%s)", valuesString)
			}

			valuesString = valuesString + "\r\n"

			return valuesString
		},
	}
	alertLink := notifiers.AlertLink(s.Notifier.UiEndpoint, alert)
	return notifiers.FormatAlert(alert, alertLink, funcMap, s.mitreStore)
}

func (*slack) Close(_ context.Context) error {
	return nil
}

// AlertNotify takes in an alert and generates the Slack message
func (s *slack) AlertNotify(ctx context.Context, alert *storage.Alert) error {
	body, err := s.getDescription(alert)
	if err != nil {
		return err
	}
	attachments := []attachment{
		{
			FallBack:       body,
			Color:          notifiers.GetAttachmentColor(alert.GetPolicy().GetSeverity()),
			Pretext:        fmt.Sprintf("*%s*", notifiers.SummaryForAlert(alert)),
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

	webhookURL := s.metadataGetter.GetAnnotationValue(ctx, alert, s.GetLabelKey(), s.GetLabelDefault())
	webhook := urlfmt.FormatURL(webhookURL, urlfmt.HTTPS, urlfmt.NoTrailingSlash)

	return retry.WithRetry(
		func() error {
			return s.postMessage(ctx, webhook, jsonPayload)
		},
		retry.OnlyRetryableErrors(),
		retry.Tries(3),
		retry.BetweenAttempts(func(previousAttempt int) {
			wait := time.Duration(previousAttempt * previousAttempt * 100)
			time.Sleep(wait * time.Millisecond)
		}),
	)
}

// NetworkPolicyYAMLNotify takes in a yaml file and generates the Slack message
func (s *slack) NetworkPolicyYAMLNotify(ctx context.Context, yaml string, clusterName string) error {
	if strings.Count(yaml, "\n") > 300 { // Looks like messages are truncated at ~340 lines.
		return errors.New("yaml is too large (>300 lines) to send over slack")
	}
	if len(yaml) > 35000 { // Slack hard limit is 40,000 characters, so leave 5,000 as a buffer to a round number.
		return errors.New("yaml is too large (>35,000 characters) to send over slack")
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
			Color:          notifiers.YAMLNotificationColor,
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
	webhook := urlfmt.FormatURL(webhookURL, urlfmt.HTTPS, urlfmt.NoTrailingSlash)

	return retry.WithRetry(
		func() error {
			return s.postMessage(ctx, webhook, jsonPayload)
		},
		retry.OnlyRetryableErrors(),
		retry.Tries(3),
		retry.BetweenAttempts(func(previousAttempt int) {
			wait := time.Duration(previousAttempt * previousAttempt * 100)
			time.Sleep(wait * time.Millisecond)
		}),
	)
}

// NewSlack exported to allow for usage in various components
func NewSlack(notifier *storage.Notifier, metadataGetter notifiers.MetadataGetter, mitreStore mitreDS.AttackReadOnlyDataStore) (*slack, error) {
	return &slack{
		Notifier: notifier,
		client: &http.Client{
			Transport: proxy.RoundTripper(),
		},
		metadataGetter: metadataGetter,
		mitreStore:     mitreStore,
	}, nil
}

func (s *slack) ProtoNotifier() *storage.Notifier {
	return s.Notifier
}

func (s *slack) Test(ctx context.Context) error {
	n := notification{
		Text: "This is a test message created to test integration with StackRox.",
	}
	jsonPayload, err := json.Marshal(&n)
	if err != nil {
		return errors.New("Could not marshal test notification")
	}

	webhook := urlfmt.FormatURL(s.GetLabelDefault(), urlfmt.HTTPS, urlfmt.NoTrailingSlash)

	return retry.WithRetry(
		func() error {
			return s.postMessage(ctx, webhook, jsonPayload)
		},
		retry.OnlyRetryableErrors(),
		retry.Tries(3),
		retry.BetweenAttempts(func(previousAttempt int) {
			wait := time.Duration(previousAttempt * previousAttempt * 100)
			time.Sleep(wait * time.Millisecond)
		}),
	)
}

func (s *slack) postMessage(ctx context.Context, url string, jsonPayload []byte) error {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req.WithContext(ctx))
	if err != nil {
		log.Errorw("Error posting message to Slack", logging.Err(err),
			logging.ErrCode(codes.SlackGeneric), logging.NotifierName(s.GetName()))
		return errors.Wrap(err, "Error posting to slack")
	}
	defer utils.IgnoreError(resp.Body.Close)

	return notifiers.CreateError(s.GetName(), resp, codes.SlackGeneric)
}
