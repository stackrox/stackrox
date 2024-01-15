package teams

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
	"github.com/stackrox/rox/central/notifiers/metadatagetter"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events/codes"
	"github.com/stackrox/rox/pkg/administration/events/option"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	and = "AND"
	or  = "OR"
)

var (
	log     = logging.LoggerForModule(option.EnableAdministrationEvents())
	timeout = env.TeamsTimeout.DurationSetting()
)

// teams notifier plugin.
type teams struct {
	*storage.Notifier

	metadataGetter notifiers.MetadataGetter
}

type section struct {
	Title string `json:"title"`
	Facts []fact `json:"facts"`
}

type fact struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// notification json struct for richly-formatted notifications.
type notification struct {
	Color    string    `json:"themeColor"`
	Title    string    `json:"title"`
	Text     string    `json:"text"`
	Sections []section `json:"sections"`
}

func (t *teams) getAlertSection(alert *storage.Alert) section {
	var facts []fact

	alertID := alert.GetId()
	if len(alertID) > 0 {
		facts = append(facts, fact{Name: "ID", Value: alertID})
	}

	alertLink := notifiers.AlertLink(t.Notifier.UiEndpoint, alert)
	if len(alertLink) > 0 {
		facts = append(facts, fact{Name: "URL", Value: alertLink})
	}

	alertTime := alert.GetTime().String()
	if len(alertTime) > 0 {
		facts = append(facts, fact{Name: "Time", Value: alertTime})
	}

	section := section{Title: "Alert Details", Facts: facts}

	policy := alert.GetPolicy()
	if policy == nil {
		return section
	}

	severityVal, err := notifiers.GetNotifiersCompatiblePolicySeverity(policy.GetSeverity().String())
	if err != nil {
		return section
	}

	section.Facts = append(facts, fact{Name: "Severity", Value: severityVal})
	return section
}

func (t *teams) getPolicySection(alert *storage.Alert) (section, error) {
	var facts []fact

	policy := alert.GetPolicy()
	if policy == nil {
		return section{}, errors.New("Policy does not exist on alert object")
	}

	if len(policy.GetDescription()) > 0 {
		facts = append(facts, fact{Name: "Description", Value: policy.GetDescription()})
	}

	if len(policy.GetRemediation()) > 0 {
		facts = append(facts, fact{Name: "Remediation", Value: policy.GetRemediation()})
	}

	if len(policy.GetRationale()) > 0 {
		facts = append(facts, fact{Name: "Rationale", Value: policy.GetRationale()})
	}

	section := section{Title: "Policy Details", Facts: facts}

	criteriaFacts := t.getSectionFacts(policy.GetPolicySections())
	section.Facts = append(section.Facts, criteriaFacts...)

	return section, nil
}

func (t *teams) getEntitySection(alert *storage.Alert) section {
	switch entity := alert.GetEntity().(type) {
	case *storage.Alert_Deployment_:
		return t.getDeploymentSection(entity.Deployment)
	case *storage.Alert_Image:
		return t.getImageSection(entity.Image)
	case *storage.Alert_Resource_:
		return t.getResourceSection(entity.Resource)
	}
	return section{}
}

func (t *teams) getResourceSection(resource *storage.Alert_Resource) section {
	facts := []fact{{Name: "Resource Name", Value: resource.GetName()},
		{Name: "Type", Value: resource.GetResourceType().String()},
		{Name: "Cluster Id", Value: resource.GetClusterId()},
		{Name: "Cluster Name", Value: resource.GetClusterName()}}

	if resource.GetNamespace() != "" {
		facts = append(facts, fact{Name: "Namespace", Value: resource.GetNamespace()})
	}

	return section{Title: "Resource Details", Facts: facts}
}

func (t *teams) getImageSection(image *storage.ContainerImage) section {
	return section{Title: "Image Details", Facts: []fact{{Name: "Image Name", Value: types.Wrapper{GenericImage: image}.FullName()}}}
}

func (t *teams) getDeploymentSection(deployment *storage.Alert_Deployment) section {
	var facts []fact

	if len(deployment.GetId()) > 0 {
		facts = append(facts, fact{Name: "ID", Value: deployment.GetId()})
	}

	if len(deployment.GetName()) > 0 {
		facts = append(facts, fact{Name: "Name", Value: deployment.GetName()})
	}

	if len(deployment.GetNamespace()) > 0 {
		facts = append(facts, fact{Name: "Namespace", Value: deployment.GetNamespace()})
	}

	if len(deployment.GetClusterId()) > 0 {
		facts = append(facts, fact{Name: "Cluster Id", Value: deployment.GetClusterId()})
	}

	if len(deployment.GetClusterName()) > 0 {
		facts = append(facts, fact{Name: "Cluster Name", Value: deployment.GetClusterName()})
	}

	deploymentContainers := deployment.GetContainers()
	var images []string
	for _, c := range deploymentContainers {
		if len(c.GetImage().GetName().GetFullName()) > 0 {
			images = append(images, c.GetImage().GetName().GetFullName())
		}
	}
	if len(images) > 0 {
		facts = append(facts, fact{Name: "Images", Value: strings.Join(images, ", ")})
	}

	return section{Title: "Deployment Details", Facts: facts}
}

func (t *teams) getViolationSection(alert *storage.Alert) (section, error) {
	var facts []fact

	violations := alert.GetViolations()
	if len(violations) == 0 {
		return section{}, errors.New("`Violations` does not exist on alert object")
	}

	for _, v := range violations {
		if len(v.GetMessage()) > 0 {
			text := fmt.Sprintf("Message : %s", v.GetMessage())
			facts = append(facts, fact{Name: "Description", Value: text})
		}
		if v.GetType() == storage.Alert_Violation_K8S_EVENT {
			for _, attr := range v.GetKeyValueAttrs().GetAttrs() {
				facts = append(facts, fact{Name: attr.GetKey(), Value: attr.GetValue()})
			}
		}
	}
	return section{Title: "Violation Details", Facts: facts}, nil
}

func (t *teams) getSectionFacts(policySections []*storage.PolicySection) []fact {
	var facts []fact
	for _, section := range policySections {
		sectionName := "Section "
		if section.GetSectionName() != "" {
			sectionName = fmt.Sprintf("%s %q", sectionName, section.GetSectionName())
		}

		groupsString := fmt.Sprintf("%s\n__________", groupsToString(section.GetPolicyGroups()))

		facts = append(facts, fact{
			Name:  sectionName,
			Value: groupsString,
		})
	}
	return facts
}

func groupsToString(groups []*storage.PolicyGroup) string {
	var groupStrings []string
	for _, group := range groups {
		var op string
		if group.GetBooleanOperator() == storage.BooleanOperator_OR {
			op = or
		} else {
			op = and
		}
		valString := valueListToString(group.GetValues(), op)
		if group.GetNegate() {
			valString = fmt.Sprintf("NOT (%s)", valString)
		}
		groupStrings = append(groupStrings, fmt.Sprintf("\n- %s: %s", group.GetFieldName(), valString))
	}
	return strings.Join(groupStrings, "")
}

func valueListToString(values []*storage.PolicyValue, opString string) string {
	var valueList []string
	for _, value := range values {
		valueList = append(valueList, value.GetValue())
	}
	joinWithWhitespace := fmt.Sprintf(" %s ", opString)
	return strings.Join(valueList, joinWithWhitespace)
}

func (*teams) Close(_ context.Context) error {
	return nil
}

// AlertNotify takes in an alert and generates the Teams message.
func (t *teams) AlertNotify(ctx context.Context, alert *storage.Alert) error {
	var sections []section
	title := notifiers.SummaryForAlert(alert)

	alertSection := t.getAlertSection(alert)
	sections = append(sections, alertSection)

	policySection, err := t.getPolicySection(alert)
	if err == nil && len(policySection.Facts) > 0 {
		sections = append(sections, policySection)
	}

	violationSection, err := t.getViolationSection(alert)
	if err == nil && len(violationSection.Facts) > 0 {
		sections = append(sections, violationSection)
	}

	entitySection := t.getEntitySection(alert)
	if len(entitySection.Facts) > 0 {
		sections = append(sections, entitySection)
	}

	notification := notification{
		Title:    title,
		Color:    notifiers.GetAttachmentColor(alert.GetPolicy().GetSeverity()),
		Text:     title,
		Sections: sections,
	}

	jsonPayload, err := json.Marshal(&notification)
	if err != nil {
		return errors.Wrapf(err, "Could not marshal notification for alert %v", alert.GetId())
	}

	webhookURL := t.metadataGetter.GetAnnotationValue(ctx, alert, t.GetLabelKey(), t.GetLabelDefault())
	webhook := urlfmt.FormatURL(webhookURL, urlfmt.HTTPS, urlfmt.NoTrailingSlash)

	return retry.WithRetry(
		func() error {
			return t.postMessage(ctx, webhook, jsonPayload)
		},
		retry.OnlyRetryableErrors(),
		retry.Tries(3),
		retry.BetweenAttempts(func(previousAttempt int) {
			time.Sleep(time.Duration(previousAttempt*previousAttempt*100) * time.Millisecond)
		}),
	)
}

// NetworkPolicyYAMLNotify takes in a yaml file and generates the teams message.
func (t *teams) NetworkPolicyYAMLNotify(ctx context.Context, yaml string, clusterName string) error {
	tagLine := fmt.Sprintf("Network policy YAML applied on cluster %q", clusterName)

	funcMap := template.FuncMap{
		"codeBlock": func(s string) string {
			if len(s) > 0 {
				return fmt.Sprintf("<pre>%s</pre>", s)
			}
			return "\n<YAML is empty>\n"
		},
	}

	body, err := notifiers.FormatNetworkPolicyYAML(yaml, clusterName, funcMap)
	if err != nil {
		return err
	}
	notification := notification{
		Title: tagLine,
		Color: notifiers.YAMLNotificationColor,
		Text:  body,
	}

	jsonPayload, err := json.Marshal(&notification)
	if err != nil {
		return errors.Wrapf(err, "Could not marshal notification for yaml for cluster %s", clusterName)
	}
	webhook := urlfmt.FormatURL(t.GetLabelDefault(), urlfmt.HTTPS, urlfmt.NoTrailingSlash)

	return retry.WithRetry(
		func() error {
			return t.postMessage(ctx, webhook, jsonPayload)
		},
		retry.OnlyRetryableErrors(),
		retry.Tries(3),
		retry.BetweenAttempts(backOff),
	)
}

// NewTeams exported to allow for usage in various components.
func NewTeams(notifier *storage.Notifier, metadataGetter notifiers.MetadataGetter) (*teams, error) {
	return &teams{
		Notifier:       notifier,
		metadataGetter: metadataGetter,
	}, nil
}

func (t *teams) ProtoNotifier() *storage.Notifier {
	return t.Notifier
}

func (t *teams) Test(ctx context.Context) *notifiers.NotifierError {
	n := notification{
		Text: "This is a test message created to test teams integration with StackRox.",
	}
	jsonPayload, err := json.Marshal(&n)
	if err != nil {
		return notifiers.NewNotifierError("create test message failed", errors.New("Could not marshal test notification"))
	}

	webhook := urlfmt.FormatURL(t.GetLabelDefault(), urlfmt.HTTPS, urlfmt.NoTrailingSlash)

	err = retry.WithRetry(
		func() error {
			return t.postMessage(ctx, webhook, jsonPayload)
		},
		retry.OnlyRetryableErrors(),
		retry.Tries(3),
		retry.BetweenAttempts(backOff),
	)

	if err != nil {
		return notifiers.NewNotifierError("send test message failed", err)
	}

	return nil
}

func (t *teams) postMessage(ctx context.Context, url string, jsonPayload []byte) error {
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{
		Timeout:   timeout,
		Transport: proxy.RoundTripper(),
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Errorw("Error posting message to teams",
			logging.Err(err),
			logging.ErrCode(codes.TeamsGeneric),
			logging.NotifierName(t.GetName()))
		return errors.Wrap(err, "Error posting to teams")
	}
	defer utils.IgnoreError(resp.Body.Close)
	return notifiers.CreateError(t.GetName(), resp, codes.TeamsGeneric)
}

func backOff(previousAttempt int) {
	time.Sleep(time.Duration(previousAttempt*previousAttempt*100) * time.Millisecond)
}

func init() {
	notifiers.Add(notifiers.TeamsType, func(notifier *storage.Notifier) (notifiers.Notifier, error) {
		s, err := NewTeams(notifier, metadatagetter.Singleton())
		return s, err
	})
}
