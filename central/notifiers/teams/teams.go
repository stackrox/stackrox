package teams

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/notifiers"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	log = logging.LoggerForModule()
)

// teams notifier plugin
type teams struct {
	*storage.Notifier
}

type section struct {
	Title string `json:"title"`
	Facts []fact `json:"facts"`
}

type fact struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// notification json struct for richly-formatted notifications
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

	alertLink := notifiers.AlertLink(t.Notifier.UiEndpoint, alertID)
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

	if policy.GetFields() != nil {
		facts := t.getPolicyFieldsFacts(reflect.ValueOf(policy.GetFields()).Elem())
		if len(facts) > 0 {
			section.Facts = append(section.Facts, facts...)
		}
	}

	return section, nil
}

func (t *teams) getDeploymentSection(alert *storage.Alert) (section, error) {
	var facts []fact
	deployment := alert.GetDeployment()
	if deployment == nil {
		return section{}, errors.New("Deployment does not exist on alert object")
	}

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

	return section{Title: "Deployment Details", Facts: facts}, nil
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
	}
	return section{Title: "Violation Details", Facts: facts}, nil
}

func (t *teams) getPolicyFieldsFacts(policyFields reflect.Value) []fact {
	var facts []fact
	for i := 0; i < policyFields.NumField(); i++ {
		fieldName := policyFields.Type().Field(i).Name
		if strings.HasPrefix(fieldName, "XXX_") {
			continue
		}
		text := translateRecursive(0, policyFields.Field(i))

		if len(text) == 0 {
			continue
		}
		facts = append(facts, fact{Name: fieldName, Value: text})
	}
	return facts
}

func translateSlice(level int, original reflect.Value) string {
	var slices []string
	switch original.Interface().(type) {
	case []string:
		for i := 0; i < original.Len(); i++ {
			ret := original.Index(i).Interface().(string)
			slices = append(slices, ret)
		}
	case []int:
		for i := 0; i < original.Len(); i++ {
			ret := strconv.Itoa(original.Index(i).Interface().(int))
			slices = append(slices, ret)
		}
	default:
		for i := 0; i < original.Len(); i++ {
			ret := translateRecursive(level, original.Field(i))
			if len(ret) > 0 {
				slices = append(slices, ret)
			}
		}
	}
	if len(slices) == 0 {
		return ""
	}
	return fmt.Sprintf("[ %s ]", strings.Join(slices, ", "))
}

func getSpacedString(count int) string {
	var builder strings.Builder
	for i := 0; i < 2*count; i++ {
		builder.WriteString(" ")
	}
	return builder.String()
}

func translateRecursive(level int, original reflect.Value) string {
	ret := ""
	spacedString := getSpacedString(level)
	switch original.Kind() {
	case reflect.Ptr:
		original = original.Elem()
		if !original.IsValid() {
			return ""
		}
		return translateRecursive(level, original)
	case reflect.Slice:
		return translateSlice(level, original)
	case reflect.Interface:
		original = original.Elem()
		return translateRecursive(level, original)
	case reflect.Struct:
		for i := 0; i < original.NumField(); i++ {
			fieldName := original.Type().Field(i).Name
			if strings.HasPrefix(fieldName, "XXX_") {
				continue
			}
			val := translateRecursive(level+1, original.Field(i))
			if len(val) > 0 {
				ret = fmt.Sprintf("%s%s - %s: %s\n", ret, spacedString, fieldName, val)
			}
		}
		if len(ret) > 0 {
			ret = fmt.Sprintf("<pre>%s</pre>", ret)
		}
	case reflect.String:
		ret = original.Interface().(string)
	case reflect.Bool:
		ret = fmt.Sprint(original.Interface().(bool))
	case reflect.Int:
		ret = strconv.Itoa(original.Interface().(int))
	case reflect.Int8:
		ret = strconv.Itoa(int(original.Interface().(int8)))
	case reflect.Int16:
		ret = strconv.Itoa(int(original.Interface().(int16)))
	case reflect.Int32:
		switch original.Interface().(type) {
		case int32:
			ret = strconv.Itoa(int(original.Interface().(int32)))
		case storage.Comparator:
			ret = storage.Comparator_name[int32(original.Interface().(storage.Comparator))]
		default:
			return ""
		}
	case reflect.Int64:
		ret = strconv.Itoa(int(original.Interface().(int64)))
	case reflect.Uint:
		ret = strconv.FormatUint(uint64(original.Interface().(uint)), 10)
	case reflect.Uint8:
		ret = strconv.FormatUint(uint64(original.Interface().(uint8)), 10)
	case reflect.Uint16:
		ret = strconv.FormatUint(uint64(original.Interface().(uint16)), 10)
	case reflect.Uint32:
		ret = strconv.FormatUint(uint64(original.Interface().(uint32)), 10)
	case reflect.Uint64:
		ret = strconv.FormatUint(original.Interface().(uint64), 10)
	case reflect.Float32:
		ret = fmt.Sprint(original.Interface().(float32))
	case reflect.Float64:
		ret = fmt.Sprint(original.Interface().(float64))
	default:
		break
	}
	return ret
}

// AlertNotify takes in an alert and generates the Teams message
func (t *teams) AlertNotify(alert *storage.Alert) error {
	var sections []section
	title := fmt.Sprintf("New Alert: Deployment %q violates %q Policy", alert.GetDeployment().GetName(), alert.GetPolicy().GetName())

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

	deploymentSection, err := t.getDeploymentSection(alert)
	if err == nil && len(deploymentSection.Facts) > 0 {
		sections = append(sections, deploymentSection)
	}

	notification := notification{
		Title:    title,
		Color:    notifiers.GetAttachmentColor(alert.GetPolicy().GetSeverity()),
		Text:     fmt.Sprintf("Deployment %q (%q) violates %q Policy", alert.GetDeployment().GetName(), alert.GetDeployment().GetId(), alert.GetPolicy().GetName()),
		Sections: sections,
	}

	jsonPayload, err := json.Marshal(&notification)
	if err != nil {
		return errors.Wrapf(err, "Could not marshal notification for alert %v", alert.GetId())
	}

	webhookURL := notifiers.GetLabelValue(alert, t.GetLabelKey(), t.GetLabelDefault())
	webhook := urlfmt.FormatURL(webhookURL, urlfmt.HTTPS, urlfmt.NoTrailingSlash)

	return retry.WithRetry(
		func() error {
			return postMessage(webhook, jsonPayload)
		},
		retry.OnlyRetryableErrors(),
		retry.Tries(3),
		retry.BetweenAttempts(func(previousAttempt int) {
			time.Sleep(time.Duration(previousAttempt*previousAttempt*100) * time.Millisecond)
		}),
	)
}

// YamlNotify takes in a yaml file and generates the teams message
func (t *teams) NetworkPolicyYAMLNotify(yaml string, clusterName string) error {
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
			return postMessage(webhook, jsonPayload)
		},
		retry.OnlyRetryableErrors(),
		retry.Tries(3),
		retry.BetweenAttempts(backOff),
	)
}

func newTeams(notifier *storage.Notifier) (*teams, error) {
	return &teams{
		Notifier: notifier,
	}, nil
}

func (t *teams) ProtoNotifier() *storage.Notifier {
	return t.Notifier
}

func (t *teams) Test() error {
	n := notification{
		Text: "This is a test message created to test teams integration with StackRox.",
	}
	jsonPayload, err := json.Marshal(&n)
	if err != nil {
		return errors.New("Could not marshal test notification")
	}

	webhook := urlfmt.FormatURL(t.GetLabelDefault(), urlfmt.HTTPS, urlfmt.NoTrailingSlash)

	return retry.WithRetry(
		func() error {
			return postMessage(webhook, jsonPayload)
		},
		retry.OnlyRetryableErrors(),
		retry.Tries(3),
		retry.BetweenAttempts(backOff),
	)
}

func postMessage(url string, jsonPayload []byte) error {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{
		Timeout:   notifiers.Timeout,
		Transport: proxy.RoundTripper(),
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Error posting to teams: %v", err)
		return errors.Wrap(err, "Error posting to teams")
	}
	defer utils.IgnoreError(resp.Body.Close)
	return notifiers.CreateError("Teams", resp)
}

func backOff(previousAttempt int) {
	time.Sleep(time.Duration(previousAttempt*previousAttempt*100) * time.Millisecond)
}

func init() {
	notifiers.Add("teams", func(notifier *storage.Notifier) (notifiers.Notifier, error) {
		s, err := newTeams(notifier)
		return s, err
	})
}
