package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	jiraLib "github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/notifiers/metadatagetter"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events/codes"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/logging"
	mitreDataStore "github.com/stackrox/rox/pkg/mitre/datastore"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	timeout = 5 * time.Second
)

var (
	log = logging.LoggerForModule()

	severities = []storage.Severity{
		storage.Severity_CRITICAL_SEVERITY,
		storage.Severity_HIGH_SEVERITY,
		storage.Severity_MEDIUM_SEVERITY,
		storage.Severity_LOW_SEVERITY,
	}

	defaultPriorities = map[storage.Severity]string{
		storage.Severity_CRITICAL_SEVERITY: "P0",
		storage.Severity_HIGH_SEVERITY:     "P1",
		storage.Severity_MEDIUM_SEVERITY:   "P2",
		storage.Severity_LOW_SEVERITY:      "P3",
	}
	pattern = regexp.MustCompile(`^(P[0-9])\b`)
)

// jira notifier plugin.
type jira struct {
	client *jiraLib.Client

	conf *storage.Jira

	notifier *storage.Notifier

	metadataGetter notifiers.MetadataGetter
	mitreStore     mitreDataStore.AttackReadOnlyDataStore

	severityToPriority map[storage.Severity]string
	needsPriority      bool

	unknownMap map[string]interface{}
}

type issueTypeResult struct {
	StartAt    int                      `json:"startAt"`
	MaxResults int                      `json:"maxResults"`
	Total      int                      `json:"total"`
	IssueTypes []*jiraLib.MetaIssueType `json:"values"`
}

func getIssueTypes(client *jiraLib.Client, project string) ([]*jiraLib.MetaIssueType, error) {
	// Low level HTTP client call is used here due to the deprecation/removal of the Jira endpoint used by the Jira library
	// to fetch the CreateMeta data, and there is no API call for the suggested endpoint to use in place of the removed one.
	// Info here:
	// https://confluence.atlassian.com/jiracore/createmeta-rest-endpoint-to-be-removed-975040986.html
	urlPath := fmt.Sprintf("rest/api/2/issue/createmeta/%s/issuetypes", project)

	req, err := client.NewRequest("GET", urlPath, nil)
	if err != nil {
		return []*jiraLib.MetaIssueType{}, err
	}

	resp, err := client.Do(req, nil)
	if err != nil {
		return []*jiraLib.MetaIssueType{}, err
	}

	result := &issueTypeResult{}
	defer utils.IgnoreError(resp.Body.Close)
	err = json.NewDecoder(resp.Body).Decode(result)
	if err != nil {
		return []*jiraLib.MetaIssueType{}, err
	}

	return result.IssueTypes, nil
}

func isPriorityNeeded(client *jiraLib.Client, project, issueType string) (bool, error) {
	issueTypes, err := getIssueTypes(client, project)
	if err != nil {
		return false, errors.Wrapf(err, "could not get meta information for JIRA project %q", project)
	}

	var validIssues []string
	for _, issue := range issueTypes {
		validIssues = append(validIssues, issue.Name)
		if !strings.EqualFold(issue.Name, issueType) {
			continue
		}
		bytes, _ := json.MarshalIndent(issue.Fields, "", "  ")
		log.Debugf("Fields for %q: %s", issue.Name, bytes)
		_, hasPriority := issue.Fields["priority"]
		return hasPriority, nil
	}
	return false, errors.Errorf("could not find issue type %q in project %q. Valid issue types are: %+v", issueType, project, validIssues)
}

func (j *jira) getAlertDescription(alert *storage.Alert) (string, error) {
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
		"section": func(s string) string {
			return fmt.Sprintf("\r\n * %v", s)
		},
		"group": func(s string) string {
			return fmt.Sprintf("\r\n ** %s", s)
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
				valueStrings = append(valueStrings, value.GetValue())
			}

			valuesString := strings.Join(valueStrings, opString)
			if negated {
				valuesString = fmt.Sprintf("NOT (%s)", valuesString)
			}

			return valuesString
		},
	}
	alertLink := notifiers.AlertLink(j.notifier.UiEndpoint, alert)
	return notifiers.FormatAlert(alert, alertLink, funcMap, j.mitreStore)
}

func (j *jira) Close(_ context.Context) error {
	return nil
}

// AlertNotify takes in an alert and generates the notification.
func (j *jira) AlertNotify(ctx context.Context, alert *storage.Alert) error {
	description, err := j.getAlertDescription(alert)
	if err != nil {
		return err
	}

	project := j.metadataGetter.GetAnnotationValue(ctx, alert, j.notifier.GetLabelKey(), j.notifier.GetLabelDefault())
	i := &jiraLib.Issue{
		Fields: &jiraLib.IssueFields{
			Summary: notifiers.SummaryForAlert(alert),
			Type: jiraLib.IssueType{
				Name: j.conf.GetIssueType(),
			},
			Project: jiraLib.Project{
				Key: project,
			},
			Description: description,
		},
	}
	err = j.createIssue(ctx, alert.GetPolicy().GetSeverity(), i)
	if err != nil {
		log.Errorw("failed to create JIRA issue for alert",
			logging.Err(err), logging.NotifierName(j.notifier.GetName()), logging.ErrCode(codes.JIRAGeneric),
			logging.AlertID(alert.GetId()))
	}
	return err
}

func (j *jira) NetworkPolicyYAMLNotify(ctx context.Context, yaml string, clusterName string) error {
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
		},
	}
	err = j.createIssue(ctx, storage.Severity_MEDIUM_SEVERITY, i)
	if err != nil {
		log.Errorw("failed to create JIRA issue for network policy",
			logging.Err(err), logging.NotifierName(j.notifier.GetName()), logging.ErrCode(codes.JIRAGeneric))
	}
	return err
}

func validate(jira *storage.Jira) error {
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
		errorList.AddString("Password or API Token must be specified")
	}

	if len(jira.GetPriorityMappings()) != 0 {
		unfoundSeverities := make(map[storage.Severity]struct{})
		for _, sev := range severities {
			unfoundSeverities[sev] = struct{}{}
		}
		for _, mapping := range jira.GetPriorityMappings() {
			delete(unfoundSeverities, mapping.GetSeverity())
		}
		for sev := range unfoundSeverities {
			errorList.AddStringf("mapping for severity %s required", sev.String())
		}
	}
	return errorList.ToError()
}

// NewJira exported to allow for usage in various components
func NewJira(notifier *storage.Notifier, metadataGetter notifiers.MetadataGetter, mitreStore mitreDataStore.AttackReadOnlyDataStore) (*jira, error) {
	conf := notifier.GetJira()
	if conf == nil {
		return nil, errors.New("Jira configuration required")
	}
	if err := validate(conf); err != nil {
		return nil, err
	}

	url := urlfmt.FormatURL(conf.GetUrl(), urlfmt.HTTPS, urlfmt.TrailingSlash)

	httpClient := &http.Client{
		Timeout: timeout,
		Transport: &jiraLib.BasicAuthTransport{
			Username:  conf.GetUsername(),
			Password:  conf.GetPassword(),
			Transport: proxy.RoundTripper(),
		},
	}

	client, err := jiraLib.NewClient(httpClient, url)
	if err != nil {
		return nil, errors.Wrap(err, "could not create JIRA client")
	}
	prios, _, err := client.Priority.GetList()
	if err != nil {
		errStr := err.Error()
		if strings.HasPrefix(errStr, "401") || strings.HasPrefix(errStr, "403") {
			httpClient := &http.Client{
				Timeout: timeout,
				Transport: &jiraLib.BearerAuthTransport{
					Token:     conf.GetPassword(),
					Transport: proxy.RoundTripper(),
				},
			}

			client, err = jiraLib.NewClient(httpClient, url)
			if err != nil {
				return nil, errors.Wrap(err, "could not create JIRA client")
			}
			prios, _, err = client.Priority.GetList()
		}
		if err != nil {
			return nil, errors.Wrap(err, "could not get the priority list")
		}
	}
	jiraConf := notifier.GetJira()

	derivedPriorities := mapPriorities(jiraConf, prios)
	if len(jiraConf.GetPriorityMappings()) == 0 {
		bytes, _ := json.Marshal(&derivedPriorities)
		log.Debugf("Derived Jira Priorities: %s", bytes)
		for k, v := range derivedPriorities {
			jiraConf.PriorityMappings = append(jiraConf.PriorityMappings, &storage.Jira_PriorityMapping{
				Severity:     k,
				PriorityName: v,
			})
		}
		sort.Slice(jiraConf.PriorityMappings, func(i, j int) bool {
			return jiraConf.PriorityMappings[i].Severity < jiraConf.PriorityMappings[j].Severity
		})
	}

	needsPriority, err := isPriorityNeeded(client, notifier.GetLabelDefault(), jiraConf.GetIssueType())
	if err != nil {
		return nil, errors.Wrapf(err, "could not determine if priority is a required field for project %q issue type %q", notifier.GetLabelDefault(), jiraConf.GetIssueType())
	}

	// marshal unknowns
	var unknownMap map[string]interface{}
	if jiraConf.GetDefaultFieldsJson() != "" {
		if err := json.Unmarshal([]byte(jiraConf.GetDefaultFieldsJson()), &unknownMap); err != nil {
			return nil, errors.Wrap(err, "could not unmarshal default fields JSON")
		}
	}

	return &jira{
		client:             client,
		conf:               notifier.GetJira(),
		notifier:           notifier,
		metadataGetter:     metadataGetter,
		mitreStore:         mitreStore,
		severityToPriority: derivedPriorities,

		needsPriority: needsPriority,
		unknownMap:    unknownMap,
	}, nil
}

func (j *jira) ProtoNotifier() *storage.Notifier {
	return j.notifier
}

func (j *jira) createIssue(_ context.Context, severity storage.Severity, i *jiraLib.Issue) error {
	i.Fields.Unknowns = j.unknownMap

	if j.needsPriority {
		i.Fields.Priority = &jiraLib.Priority{
			Name: j.severityToPriority[severity],
		}
	}

	_, resp, err := j.client.Issue.Create(i)
	if err != nil && resp == nil {
		return errors.Errorf("Error creating issue. Response: %v", err)
	}
	if err != nil {
		bytes, readErr := io.ReadAll(resp.Body)
		if readErr == nil {
			return errors.Wrapf(err, "error creating issue. Response: %s", bytes)
		}
	}
	return err
}

func (j *jira) Test(ctx context.Context) error {
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
		},
	}
	return j.createIssue(ctx, storage.Severity_LOW_SEVERITY, i)
}

// Optimistically tries to match all of the Jira priorities with the known mapping defined in defaultPriorities
// If any severity is not matched, then it returns a nil map
func optimisticMatching(prios []jiraLib.Priority) map[storage.Severity]string {
	shortened := make(map[string]string)
	for _, prio := range prios {
		if match := pattern.FindString(prio.Name); len(match) > 0 {
			shortened[match] = prio.Name
		}
	}
	output := make(map[storage.Severity]string)
	for k, name := range defaultPriorities {
		match, ok := shortened[name]
		if !ok {
			return nil
		}
		output[k] = match
	}
	return output
}

func mapPriorities(integration *storage.Jira, prios []jiraLib.Priority) map[storage.Severity]string {
	// Prioritize the defined mappings, which based on validation must contain mappings for ALL severities
	if len(integration.GetPriorityMappings()) != 0 {
		priorities := make(map[storage.Severity]string)
		for _, mapping := range integration.GetPriorityMappings() {
			priorities[mapping.GetSeverity()] = mapping.GetPriorityName()
		}
		return priorities
	}
	if matching := optimisticMatching(prios); matching != nil {
		return matching
	}
	// Lexicographically sort the priorities retrieved from Jira, which as far as we know are
	// single digit IDs in string form. It's possible that the Jira installation has fewer priorities than our
	// severities and therefore we will attribute the last priority from Jira to the remaining severities
	sort.Slice(prios, func(i, j int) bool {
		numI, errI := strconv.Atoi(prios[i].ID)
		numJ, errJ := strconv.Atoi(prios[j].ID)
		if errI == nil && errJ == nil {
			return numI < numJ
		}
		if errI != errJ {
			return errI == nil // all numeric before all non-numeric
		}
		return prios[i].ID < prios[j].ID
	})
	// Truncate priorities to the number of severities
	if len(prios) > len(severities) {
		prios = prios[:len(severities)]
	}
	priorities := make(map[storage.Severity]string)
	for i, sev := range severities {
		if i > len(prios)-1 {
			priorities[sev] = prios[len(prios)-1].Name
			continue
		}
		priorities[sev] = prios[i].Name
	}
	return priorities
}

func init() {
	notifiers.Add(notifiers.JiraType, func(notifier *storage.Notifier) (notifiers.Notifier, error) {
		j, err := NewJira(notifier, metadatagetter.Singleton(), mitreDataStore.Singleton())
		return j, err
	})
}
