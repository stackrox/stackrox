package jira

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"text/template"
	"time"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/notifications/notifiers"
	"bitbucket.org/stack-rox/apollo/pkg/notifications/types"
	"bitbucket.org/stack-rox/apollo/pkg/urlfmt"
	jiraLib "github.com/andygrunwald/go-jira"
)

const (
	timeout = 5 * time.Second
)

var (
	log = logging.New("notifiers/jira")
)

// Jira notifier plugin
type jira struct {
	client *jiraLib.Client

	username  string
	project   string
	issueType string

	*v1.Notifier
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
	alertLink := notifiers.AlertLink(j.Notifier.UiEndpoint, alert.GetId())
	return notifiers.FormatPolicy(alert, alertLink, funcMap)
}

func (j *jira) getBenchmarkDescription(schedule *v1.BenchmarkSchedule) (string, error) {
	benchmarkLink := notifiers.BenchmarkLink(j.Notifier.UiEndpoint)
	return notifiers.FormatBenchmark(schedule, benchmarkLink)
}

// AlertNotify takes in an alert and generates the notification
func (j *jira) AlertNotify(alert *v1.Alert) error {
	description, err := j.getAlertDescription(alert)
	if err != nil {
		return err
	}

	i := &jiraLib.Issue{
		Fields: &jiraLib.IssueFields{
			Summary: fmt.Sprintf("Deployment %v (%v) violates '%v' Policy", alert.Deployment.Name, alert.Deployment.Id, alert.Policy.Name),
			Type: jiraLib.IssueType{
				Name: j.issueType,
			},
			Project: jiraLib.Project{
				Key: j.project,
			},
			Description: description,
			Priority: &jiraLib.Priority{
				Name: severityToPriority(alert.GetPolicy().GetSeverity()),
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
			Summary: fmt.Sprintf("New Benchmark Results for %v", schedule.GetName()),
			Type: jiraLib.IssueType{
				Name: j.issueType,
			},
			Project: jiraLib.Project{
				Key: j.project,
			},
			Description: description,
			Priority: &jiraLib.Priority{
				Name: "P3-Low",
			},
		},
	}
	return j.createIssue(i)
}

func newJira(protoNotifier *v1.Notifier) (*jira, error) {
	username, ok := protoNotifier.Config["username"]
	if !ok {
		return nil, fmt.Errorf("username must be defined in the Jira Configuration")
	}
	password, ok := protoNotifier.Config["password"]
	if !ok {
		return nil, fmt.Errorf("password must be defined in the Jira Configuration")
	}
	project, ok := protoNotifier.Config["project"]
	if !ok {
		return nil, fmt.Errorf("project must be defined in the Jira Configuration")
	}
	issueType, ok := protoNotifier.Config["issue_type"]
	if !ok {
		return nil, fmt.Errorf("issue_type must be defined in the Jira Configuration")
	}
	url, ok := protoNotifier.Config["url"]
	if !ok {
		return nil, fmt.Errorf("url must be defined in the Jira Configuration")
	}

	url, err := urlfmt.FormatURL(url, true, true)
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
	res, err := client.Authentication.AcquireSessionCookie(username, password)
	if err != nil {
		return nil, err
	}
	if !res {
		return nil, errors.New("Result of authentication is false")
	}
	// forces the auth to use basic auth per request
	client.Authentication.SetBasicAuth(username, password)

	return &jira{
		client:    client,
		Notifier:  protoNotifier,
		project:   project,
		username:  username,
		issueType: issueType,
	}, nil
}

func (j *jira) ProtoNotifier() *v1.Notifier {
	return j.Notifier
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
				Name: j.issueType,
			},
			Project: jiraLib.Project{
				Key: j.project,
			},
			Summary: "This is a test issue created to test integration with StackRox.",
			Priority: &jiraLib.Priority{
				Name: severityToPriority(v1.Severity_LOW_SEVERITY),
			},
		},
	}
	return j.createIssue(i)
}

func severityToPriority(sev v1.Severity) string {
	switch sev {
	case v1.Severity_CRITICAL_SEVERITY:
		return "P0-Highest"
	case v1.Severity_HIGH_SEVERITY:
		return "P1-High"
	case v1.Severity_MEDIUM_SEVERITY:
		return "P2-Medium"
	case v1.Severity_LOW_SEVERITY:
		return "P3-Low"
	default:
		return "P4-Lowest"
	}
}

func init() {
	notifiers.Add("jira", func(notifier *v1.Notifier) (types.Notifier, error) {
		j, err := newJira(notifier)
		return j, err
	})
}
