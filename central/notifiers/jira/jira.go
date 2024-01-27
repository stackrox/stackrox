package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	jiraLib "github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/notifiers/metadatagetter"
	notifierUtils "github.com/stackrox/rox/central/notifiers/utils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events/codes"
	"github.com/stackrox/rox/pkg/cryptoutils/cryptocodec"
	"github.com/stackrox/rox/pkg/env"
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
)

// jira notifier plugin.
type jira struct {
	client   *jiraLib.Client
	conf     *storage.Jira
	notifier *storage.Notifier

	metadataGetter notifiers.MetadataGetter
	mitreStore     mitreDataStore.AttackReadOnlyDataStore

	severityToPriority map[storage.Severity]string
	needsPriority      bool

	unknownMap map[string]interface{}
}

type issueTypeResult struct {
	StartAt         int                      `json:"startAt"`
	MaxResults      int                      `json:"maxResults"`
	Total           int                      `json:"total"`
	IssueTypes      []*jiraLib.MetaIssueType `json:"values"`
	IssueTypesCloud []*jiraLib.MetaIssueType `json:"issueTypes"`
}

type issueField struct {
	Name    string `json:"name"`
	Key     string `json:"key"`
	FieldID string `json:"fieldId"`
}

type issueFieldsResult struct {
	StartAt          int           `json:"startAt"`
	MaxResults       int           `json:"maxResults"`
	Total            int           `json:"total"`
	IssueFields      []*issueField `json:"values"`
	IssueFieldsCloud []*issueField `json:"fields"`
}

type permissionResult struct {
	Permissions map[string]struct {
		HavePermission bool
	}
}

func callJira(client *jiraLib.Client, urlPath string, result interface{}, startAt int) error {
	req, err := client.NewRequest("GET", urlPath, nil)
	if err != nil {
		return errors.Wrap(err, "Failed to create Jira request")
	}

	values := req.URL.Query()
	values.Set("startAt", strconv.Itoa(startAt))
	req.URL.RawQuery = values.Encode()

	resp, err := client.Do(req, nil)
	if err != nil {
		return errors.Wrap(err, "Failed to successfully make request to Jira")
	}

	defer utils.IgnoreError(resp.Body.Close)
	err = json.NewDecoder(resp.Body).Decode(result)
	if err != nil {
		return errors.Wrap(err, "Failed to decode JSON response from Jira")
	}

	return nil
}

func getIssueTypes(client *jiraLib.Client, project string) ([]*jiraLib.MetaIssueType, error) {
	urlPath := fmt.Sprintf("rest/api/2/issue/createmeta/%s/issuetypes", project)

	var result issueTypeResult

	err := callJira(client, urlPath, &result, 0)

	if err != nil {
		return []*jiraLib.MetaIssueType{}, err
	}

	returnList := make([]*jiraLib.MetaIssueType, 0, result.Total)

	if len(result.IssueTypes) == 0 {
		returnList = append(returnList, result.IssueTypesCloud...)
	} else {
		returnList = append(returnList, result.IssueTypes...)
	}

	for len(returnList) < result.Total {
		result = issueTypeResult{}
		err = callJira(client, urlPath, &result, len(returnList))
		if err != nil {
			return nil, err
		}

		var actualIssueTypes []*jiraLib.MetaIssueType
		if len(result.IssueTypes) == 0 {
			actualIssueTypes = result.IssueTypesCloud
		} else {
			actualIssueTypes = result.IssueTypes
		}

		returnList = append(returnList, actualIssueTypes...)
	}

	return returnList, nil
}

func getIssueFields(client *jiraLib.Client, project, issueID string) ([]*issueField, error) {
	urlPath := fmt.Sprintf("rest/api/2/issue/createmeta/%s/issuetypes/%s", project, issueID)

	var result issueFieldsResult

	err := callJira(client, urlPath, &result, 0)
	if err != nil {
		return nil, err
	}

	returnList := make([]*issueField, 0, result.Total)

	if len(result.IssueFields) == 0 {
		returnList = append(returnList, result.IssueFieldsCloud...)
	} else {
		returnList = append(returnList, result.IssueFields...)
	}

	for len(returnList) < result.Total {
		result = issueFieldsResult{}
		err = callJira(client, urlPath, &result, len(returnList))
		if err != nil {
			return nil, err
		}

		var actualIssueTypes []*issueField
		if len(result.IssueFields) == 0 {
			actualIssueTypes = result.IssueFieldsCloud
		} else {
			actualIssueTypes = result.IssueFields
		}

		returnList = append(returnList, actualIssueTypes...)
	}

	return returnList, nil
}

func isPriorityFieldOnIssueType(client *jiraLib.Client, project, issueType string) (bool, error) {
	// Get issue types

	// Low level HTTP client call is used here due to the deprecation/removal of the Jira endpoint used by the Jira library
	// to fetch the CreateMeta data, and there is no API call for the suggested endpoint to use in place of the removed one.
	// Info here:
	// https://confluence.atlassian.com/jiracore/createmeta-rest-endpoint-to-be-removed-975040986.html
	issueTypes, err := getIssueTypes(client, project)
	if err != nil {
		return false, errors.Wrapf(err, "could not get meta information for JIRA project %q", project)
	}

	// Validate that the desired type exists and get its ID
	var issueID string
	for _, issue := range issueTypes {
		if strings.EqualFold(issue.Name, issueType) {
			issueID = issue.Id
		}
	}

	if issueID == "" {
		return false, errors.Errorf("could not find issue type %q in project %q.", issueType, project)
	}

	// Fetch its fields
	issueTypeFields, err := getIssueFields(client, project, issueID)
	if err != nil {
		return false, errors.Wrapf(err, "could not get meta information for JIRA project %q and issue type %s", project, issueType)
	}

	// Validate priority is one of the fields
	for _, field := range issueTypeFields {
		if strings.EqualFold("priority", field.Name) {
			return true, nil
		}
	}

	return false, nil
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

// Validate Jira notifier
func Validate(jira *storage.Jira, validateSecret bool) error {
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
	if validateSecret && jira.GetPassword() == "" {
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

func newJira(notifier *storage.Notifier, metadataGetter notifiers.MetadataGetter, mitreStore mitreDataStore.AttackReadOnlyDataStore,
	cryptoCodec cryptocodec.CryptoCodec, cryptoKey string) (*jira, error) {
	conf := notifier.GetJira()
	if conf == nil {
		return nil, errors.New("Jira configuration required")
	}
	if err := Validate(conf, !env.EncNotifierCreds.BooleanSetting()); err != nil {
		return nil, err
	}

	client, err := createClient(notifier, cryptoCodec, cryptoKey)

	if err != nil {
		return nil, err
	}

	canCreateIssues, err := canCreateIssuesInProject(client, notifier.GetLabelDefault())

	if err != nil {
		return nil, err
	}

	if !canCreateIssues {
		return nil, fmt.Errorf("Cannot create issues in project %s", notifier.GetLabelDefault())
	}

	var priorityMapping map[storage.Severity]string
	if !conf.DisablePriority {
		priorityMapping, err = configurePriority(client, conf, notifier.GetLabelDefault())

		if err != nil {
			return nil, err
		}
	}

	// marshal unknowns
	var unknownMap map[string]interface{}
	if conf.GetDefaultFieldsJson() != "" {
		if err := json.Unmarshal([]byte(conf.GetDefaultFieldsJson()), &unknownMap); err != nil {
			return nil, errors.Wrap(err, "could not unmarshal default fields JSON")
		}
	}

	return &jira{
		client:             client,
		conf:               notifier.GetJira(),
		notifier:           notifier,
		metadataGetter:     metadataGetter,
		mitreStore:         mitreStore,
		severityToPriority: priorityMapping,

		needsPriority: !conf.DisablePriority,
		unknownMap:    unknownMap,
	}, nil
}

func createClient(notifier *storage.Notifier, cryptoCodec cryptocodec.CryptoCodec, cryptoKey string) (*jiraLib.Client, error) {
	var (
		err  error
		resp *jiraLib.Response
		req  *http.Request
	)

	conf := notifier.GetJira()
	decCreds := conf.GetPassword()

	if env.EncNotifierCreds.BooleanSetting() {
		decCreds, err = cryptoCodec.Decrypt(cryptoKey, notifier.GetNotifierSecret())
		if err != nil {
			return nil, errors.Errorf("Error decrypting notifier secret for notifier '%s'", notifier.GetName())
		}
	}

	url := urlfmt.FormatURL(conf.GetUrl(), urlfmt.HTTPS, urlfmt.TrailingSlash)

	httpClient := &http.Client{
		Timeout: timeout,
		Transport: &jiraLib.BasicAuthTransport{
			Username:  conf.GetUsername(),
			Password:  decCreds,
			Transport: proxy.RoundTripper(),
		},
	}

	client, err := jiraLib.NewClient(httpClient, url)
	if err != nil {
		return nil, errors.Wrap(err, "could not create JIRA client")
	}

	// Test auth to Jira
	urlPath := "rest/api/2/configuration"
	if req, err = client.NewRequest("GET", urlPath, nil); err != nil {
		return nil, errors.Wrap(err, "could not create request to Jira")
	}

	log.Debugf("Making request to Jira at %s", urlPath)
	if resp, err = client.Do(req, nil); err != nil {
		// If the underlying http.Client.Do() returns an error, the Jira response will be nil.
		if resp == nil || (resp.StatusCode != 401 && resp.StatusCode != 403) {
			return nil, errors.Wrap(err, "Could not make request to Jira")
		}
		log.Debug("Retrying request to Jira using Bearer auth")
		httpClient = &http.Client{
			Timeout: timeout,
			Transport: &jiraLib.BearerAuthTransport{
				Token:     decCreds,
				Transport: proxy.RoundTripper(),
			},
		}
		if client, err = jiraLib.NewClient(httpClient, url); err != nil {
			return nil, errors.Wrap(err, "could not create Jira client with bearer auth")
		}
		if req, err = client.NewRequest("GET", urlPath, nil); err != nil {
			return nil, errors.Wrap(err, "could not create request to Jira")
		}
		if _, err = client.Do(req, nil); err != nil {
			return nil, errors.Wrap(err, "Could not make authenticated request to Jira")
		}
		log.Debug("Successfully made request to jira using bearer auth")
	}

	return client, nil
}

func canCreateIssuesInProject(client *jiraLib.Client, project string) (bool, error) {
	urlPath := fmt.Sprintf("rest/api/2/mypermissions/?projectKey=%s&permissions=CREATE_ISSUES", project)

	req, err := client.NewRequest("GET", urlPath, nil)
	if err != nil {
		return false, err
	}

	log.Debugf("Making request to %s", urlPath)
	resp, err := client.Do(req, nil)
	if err != nil {
		log.Debugf("Raw error message from jira lib: %s", err.Error())
		if resp != nil && resp.StatusCode == 404 {
			return false, fmt.Errorf("Project %s not found", project)
		}
		return false, err
	}

	result := &permissionResult{}

	defer utils.IgnoreError(resp.Body.Close)
	err = json.NewDecoder(resp.Body).Decode(result)
	if err != nil {
		return false, err
	}

	return result.Permissions["CREATE_ISSUES"].HavePermission, nil
}

func configurePriority(client *jiraLib.Client, jiraConf *storage.Jira, project string) (map[storage.Severity]string, error) {
	hasPriority, err := isPriorityFieldOnIssueType(client, project, jiraConf.GetIssueType())
	if err != nil {
		return nil, errors.Wrapf(err, "could not determine if priority is a required field for project %q issue type %q", project, jiraConf.GetIssueType())
	}

	if !hasPriority {
		errMsg := "Priority field not found on requested issue type %s in project %s. Consider checking the 'Disable setting priority' box."
		return nil, fmt.Errorf(errMsg, jiraConf.GetIssueType(), project)
	}

	prios, _, err := client.Priority.GetList()
	if err != nil {
		return nil, errors.Wrap(err, "could not get the priority list")
	}

	return mapPriorities(prios, jiraConf.GetPriorityMappings())
}

func mapPriorities(prios []jiraLib.Priority, storageMapping []*storage.Jira_PriorityMapping) (map[storage.Severity]string, error) {
	if len(storageMapping) == 0 {
		return nil, errors.New("Please define priority mappings")
	}

	prioNameSet := map[string]string{}
	for _, prio := range prios {
		prioNameSet[prio.Name] = ""
	}

	finalizedMapping := map[storage.Severity]string{}
	missingFromJira := []string{}
	for _, prioMapping := range storageMapping {
		if _, exists := prioNameSet[prioMapping.PriorityName]; exists {
			finalizedMapping[prioMapping.Severity] = prioMapping.PriorityName
		} else {
			missingFromJira = append(missingFromJira, prioMapping.PriorityName)
		}
	}

	if len(missingFromJira) > 0 {
		return nil, fmt.Errorf("Priority mappings that do not exist in Jira: %v", missingFromJira)
	}

	return finalizedMapping, nil
}

func (j *jira) ProtoNotifier() *storage.Notifier {
	return j.notifier
}

func (j *jira) createIssue(_ context.Context, severity storage.Severity, i *jiraLib.Issue) error {
	i.Fields.Unknowns = j.unknownMap

	if !j.conf.DisablePriority {
		i.Fields.Priority = &jiraLib.Priority{
			Name: j.severityToPriority[severity],
		}
	}

	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(i)
	if err != nil {
		return err
	}
	log.Debug(buf)

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

func (j *jira) Test(ctx context.Context) *notifiers.NotifierError {
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

	if err := j.createIssue(ctx, storage.Severity_LOW_SEVERITY, i); err != nil {
		return notifiers.NewNotifierError("create test Jira issue failed", err)
	}

	return nil
}

func init() {
	cryptoKey := ""
	var err error
	if env.EncNotifierCreds.BooleanSetting() {
		cryptoKey, _, err = notifierUtils.GetActiveNotifierEncryptionKey()
		if err != nil {
			utils.Should(errors.Wrap(err, "Error reading encryption key, notifier will be unable to send notifications"))
		}
	}
	notifiers.Add(notifiers.JiraType, func(notifier *storage.Notifier) (notifiers.Notifier, error) {
		j, err := newJira(notifier, metadatagetter.Singleton(), mitreDataStore.Singleton(), cryptocodec.Singleton(), cryptoKey)
		return j, err
	})
}
