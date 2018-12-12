package jira

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"text/template"
	"time"

	jiraLib "github.com/andygrunwald/go-jira"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/urlfmt"
)

const (
	timeout = 5 * time.Second
)

var (
	log = logging.LoggerForModule()
)

// Jira notifier plugin
type jira struct {
	client *jiraLib.Client

	conf *v1.Jira

	notifier *v1.Notifier
}

func (j *jira) getAlertDescription(alert *v1.Alert) (string, error) {
	funcMap := template.FuncMap{
		"header": func(s string) string {
			return fmt.Sprintf("\r\n h4. %v\r\n", s)
		},
		"subheader": func(s string) string {
			return fmt.Sprintf("\r\n h5. %v\r\n", s)
		},
		"line": func(s string) string {
			return fmt.Sprintf("%v\r\n", s)
		},
		"list": func(s string) string {
			return fmt.Sprintf("* %v\r\n", s)
		},
		"nestedList": func(s string) string {
			return fmt.Sprintf("** %v\r\n", s)
		},
	}
	alertLink := notifiers.AlertLink(j.notifier.UiEndpoint, alert.GetId())
	return notifiers.FormatPolicy(alert, alertLink, funcMap)
}

func (j *jira) getBenchmarkDescription(schedule *v1.BenchmarkSchedule) (string, error) {
	benchmarkLink := notifiers.BenchmarkLink(j.notifier.UiEndpoint)
	return notifiers.FormatBenchmark(schedule, benchmarkLink)
}

// AlertNotify takes in an alert and generates the notification
func (j *jira) AlertNotify(alert *v1.Alert) error {
	description, err := j.getAlertDescription(alert)
	if err != nil {
		return err
	}

	project := notifiers.GetLabelValue(alert, j.notifier.GetLabelKey(), j.notifier.GetLabelDefault())
	i := &jiraLib.Issue{
		Fields: &jiraLib.IssueFields{
			Summary: fmt.Sprintf("Deployment %v (%v) violates '%v' Policy", alert.Deployment.Name, alert.Deployment.Id, alert.Policy.Name),
			Type: jiraLib.IssueType{
				Name: j.conf.GetIssueType(),
			},
			Project: jiraLib.Project{
				Key: project,
			},
			Description: description,
			Priority: &jiraLib.Priority{
				Name: severityToPriority(alert.GetPolicy().GetSeverity()),
			},
		},
	}
	return j.createIssue(i)
}

func (j *jira) NetworkPolicyYAMLNotify(yaml string, clusterName string) error {
	funcMap := template.FuncMap{
		"codeBlock": func(s string) string {
			return fmt.Sprintf("{code:title=Network Policy YAML|theme=FadeToGrey|language=yaml}%s{code}", s)
		},
	}

	description, err := notifiers.FormatNetworkPolicyYAML(yaml, clusterName, funcMap)
	if err != nil {
		return err
	}

	project := j.notifier.GetLabelDefault()
	i := &jiraLib.Issue{
		Fields: &jiraLib.IssueFields{
			Summary: fmt.Sprintf("Network policy yaml to apply on cluster %s", clusterName),
			Type: jiraLib.IssueType{
				Name: j.conf.GetIssueType(),
			},
			Project: jiraLib.Project{
				Key: project,
			},
			Description: description,
			Priority: &jiraLib.Priority{
				Name: severityToPriority(storage.Severity_MEDIUM_SEVERITY),
			},
		},
	}
	return j.createIssue(i)
}

// BenchmarkNotify takes in a benchmark and generates the notification
func (j *jira) BenchmarkNotify(schedule *v1.BenchmarkSchedule) error {
	description, err := j.getBenchmarkDescription(schedule)
	if err != nil {
		return err
	}

	i := &jiraLib.Issue{
		Fields: &jiraLib.IssueFields{
			Summary: fmt.Sprintf("New Benchmark Results for %v", schedule.GetBenchmarkName()),
			Type: jiraLib.IssueType{
				Name: j.conf.GetIssueType(),
			},
			Project: jiraLib.Project{
				Key: j.notifier.GetLabelDefault(),
			},
			Description: description,
			Priority: &jiraLib.Priority{
				Name: severityToPriority(storage.Severity_LOW_SEVERITY),
			},
		},
	}
	return j.createIssue(i)
}

func validate(jira *v1.Jira) error {
	errorList := errorhelpers.NewErrorList("Jira validation")
	if jira.GetIssueType() == "" {
		errorList.AddString("Issue Type must be specified")
	}
	if jira.GetUrl() == "" {
		errorList.AddString("URL must be specified")
	}
	if jira.GetUsername() == "" {
		errorList.AddString("Username must be specified")
	}
	if jira.GetPassword() == "" {
		errorList.AddString("Password must be specified")
	}
	return errorList.ToError()
}

func newJira(notifier *v1.Notifier) (*jira, error) {
	jiraConfig, ok := notifier.GetConfig().(*v1.Notifier_Jira)
	if !ok {
		return nil, fmt.Errorf("Jira configuration required")
	}
	conf := jiraConfig.Jira
	if err := validate(conf); err != nil {
		return nil, err
	}

	url, err := urlfmt.FormatURL(conf.GetUrl(), urlfmt.HTTPS, urlfmt.TrailingSlash)
	if err != nil {
		return nil, err
	}
	httpClient := &http.Client{
		Timeout: timeout,
	}
	client, err := jiraLib.NewClient(httpClient, url)
	if err != nil {
		return nil, err
	}
	res, err := client.Authentication.AcquireSessionCookie(conf.GetUsername(), conf.GetPassword())
	if err != nil {
		return nil, err
	}
	if !res {
		return nil, errors.New("Result of authentication is false")
	}
	// forces the auth to use basic auth per request
	client.Authentication.SetBasicAuth(conf.GetUsername(), conf.GetPassword())

	return &jira{
		client:   client,
		conf:     notifier.GetConfig().(*v1.Notifier_Jira).Jira,
		notifier: notifier,
	}, nil
}

func (j *jira) ProtoNotifier() *v1.Notifier {
	return j.notifier
}

func (j *jira) createIssue(i *jiraLib.Issue) error {
	_, resp, err := j.client.Issue.Create(i)
	if err != nil {
		bytes, readErr := ioutil.ReadAll(resp.Body)
		if readErr == nil {
			return fmt.Errorf("Error creating issue: %+v. Response: %v", err, string(bytes))
		}
	}
	return err
}

func (j *jira) Test() error {
	i := &jiraLib.Issue{
		Fields: &jiraLib.IssueFields{
			Description: "StackRox Test Issue",
			Type: jiraLib.IssueType{
				Name: j.conf.GetIssueType(),
			},
			Project: jiraLib.Project{
				Key: j.notifier.GetLabelDefault(),
			},
			Summary: "This is a test issue created to test integration with StackRox.",
			Priority: &jiraLib.Priority{
				Name: severityToPriority(storage.Severity_LOW_SEVERITY),
			},
		},
	}
	return j.createIssue(i)
}

func severityToPriority(sev storage.Severity) string {
	switch sev {
	case storage.Severity_CRITICAL_SEVERITY:
		return "P0-Highest"
	case storage.Severity_HIGH_SEVERITY:
		return "P1-High"
	case storage.Severity_MEDIUM_SEVERITY:
		return "P2-Medium"
	case storage.Severity_LOW_SEVERITY:
		return "P3-Low"
	default:
		return "P4-Lowest"
	}
}

func init() {
	notifiers.Add("jira", func(notifier *v1.Notifier) (notifiers.Notifier, error) {
		j, err := newJira(notifier)
		return j, err
	})
}
