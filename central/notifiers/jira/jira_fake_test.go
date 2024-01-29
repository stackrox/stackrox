package jira

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	jiraLib "github.com/andygrunwald/go-jira"
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cryptoutils/cryptocodec"
	mitreMocks "github.com/stackrox/rox/pkg/mitre/datastore/mocks"
	notifierMocks "github.com/stackrox/rox/pkg/notifiers/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// fakeJira is a fake JIRA backend that implements exactly the APIs that the JIRA notifier needs (and only to the extent
// required by the notifier code).
// This is in no way intended to be a realistic model of the JIRA API, it only allows us to exercise notifier code paths
// in this test.
type fakeJira struct {
	cloud                     bool
	t                         *testing.T
	username, password, token string

	priorities []jiraLib.Priority
	project    jiraLib.MetaProject

	createdIssues []jiraLib.Issue
}

func (j *fakeJira) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/2/configuration", j.handleConfiguration)
	mux.HandleFunc("/rest/api/2/mypermissions/", j.handleMyPermissions)
	mux.HandleFunc("/rest/api/2/priority", j.handlePriority)
	mux.HandleFunc("/rest/api/2/issue/createmeta/FJ/issuetypes", j.handleIssueType)
	mux.HandleFunc("/rest/api/2/issue/createmeta/FJ/issuetypes/25", j.handleIssueTypeFields)
	mux.HandleFunc("/rest/api/2/issue", j.handleCreateIssue)

	if j.username == "" && j.password == "" {
		return mux
	}

	basicAuthHeader := fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", j.username, j.password))))
	tokenAuthHeader := fmt.Sprintf("Bearer %s", j.token)
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Authorization") != basicAuthHeader && req.Header.Get("Authorization") != tokenAuthHeader {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		mux.ServeHTTP(w, req)
	})
}

func (j *fakeJira) handleConfiguration(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(200)
}

func (j *fakeJira) handleIssueTypeFields(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	targetIssueType := j.project.GetIssueTypeWithName("IssueWithPrio")
	result := issueFieldsResult{
		Total: len(targetIssueType.Fields),
	}

	iFields := []*issueField{{
		Name: "Priority",
	}}

	if j.cloud {
		result.IssueFieldsCloud = iFields
	} else {
		result.IssueFields = iFields
	}

	require.NoError(j.t, json.NewEncoder(w).Encode(result))
}

func (j *fakeJira) handleMyPermissions(w http.ResponseWriter, r *http.Request) {
	if projectKey := r.URL.Query().Get("projectKey"); projectKey == "" {
		w.WriteHeader(404)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	require.NoError(j.t, json.NewEncoder(w).Encode(permissionResult{
		Permissions: map[string]struct {
			HavePermission bool
		}{
			"CREATE_ISSUES": {HavePermission: true},
		},
	}))
}

func (j *fakeJira) handlePriority(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	require.NoError(j.t, json.NewEncoder(w).Encode(j.priorities))
}

func (j *fakeJira) handleIssueType(w http.ResponseWriter, req *http.Request) {
	pathSuffix, found := strings.CutPrefix(req.URL.Path, "/rest/api/2/issue/createmeta/")

	if !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	projectKey := strings.Split(pathSuffix, "/")

	if projectKey[0] != j.project.Key || projectKey[1] != "issuetypes" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	issueTypes := &issueTypeResult{
		Total: len(j.project.IssueTypes),
	}

	if j.cloud {
		issueTypes.IssueTypesCloud = j.project.IssueTypes
	} else {
		issueTypes.IssueTypes = j.project.IssueTypes
	}

	require.NoError(j.t, json.NewEncoder(w).Encode(&issueTypes))
}

func (j *fakeJira) handleCreateIssue(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var issue jiraLib.Issue
	if !assert.NoError(j.t, json.NewDecoder(req.Body).Decode(&issue)) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	j.createdIssues = append(j.createdIssues, issue)

	w.Header().Set("Content-Type", "application/json")
	require.NoError(j.t, json.NewEncoder(w).Encode(&issue))
}

func TestWithFakeJira(t *testing.T) {
	testWithFakeJira(t, false)
}

func TestWithFakeJiraCloud(t *testing.T) {
	testWithFakeJira(t, true)
}

func testWithFakeJira(t *testing.T, cloud bool) {
	const (
		username = "fakejirauser"
		password = "fakejirapassword"
		token    = "faketoken"

		projectKey = "FJ"
	)

	priorities := []jiraLib.Priority{
		{
			Name: "P0",
			ID:   "1",
		},
		{
			Name: "P1",
			ID:   "2",
		},
		{
			Name: "P2",
			ID:   "3",
		},
		{
			Name: "P4",
			ID:   "4",
		},
		{
			Name: "P3",
			ID:   "5",
		},
	}

	project := jiraLib.MetaProject{
		Name: "FakeJira Project",
		Key:  projectKey,
		IssueTypes: []*jiraLib.MetaIssueType{
			{
				Name: "IssueWithoutPrio",
				Id:   "24",
			},
			{
				Name: "IssueWithPrio",
				Id:   "25",
				Fields: map[string]interface{}{
					"priority": true,
				},
			},
		},
	}

	fj := fakeJira{
		cloud:      cloud,
		t:          t,
		username:   username,
		password:   password,
		token:      token,
		priorities: priorities,
		project:    project,
	}

	testSrv := httptest.NewServer(fj.Handler())
	defer testSrv.Close()

	fakeJiraStorageConfig := storage.Jira{
		Url:       testSrv.URL,
		Username:  "fakejirauser",
		Password:  "badpassword",
		IssueType: "IssueWithPrio",
		PriorityMappings: []*storage.Jira_PriorityMapping{
			{
				Severity:     storage.Severity_CRITICAL_SEVERITY,
				PriorityName: "P0",
			},
			{
				Severity:     storage.Severity_HIGH_SEVERITY,
				PriorityName: "P1",
			},
			{
				Severity:     storage.Severity_MEDIUM_SEVERITY,
				PriorityName: "P2",
			},
			{
				Severity:     storage.Severity_LOW_SEVERITY,
				PriorityName: "P3",
			},
		},
	}
	fakeJiraConfig := &storage.Notifier{
		Name:         "FakeJIRA",
		UiEndpoint:   "https://central.stackrox",
		Type:         "jira",
		LabelDefault: projectKey,
		Config: &storage.Notifier_Jira{
			Jira: &fakeJiraStorageConfig,
		},
	}

	mockCtrl := gomock.NewController(t)
	mitreStore := mitreMocks.NewMockAttackReadOnlyDataStore(mockCtrl)
	metadataGetter := notifierMocks.NewMockMetadataGetter(mockCtrl)
	metadataGetter.EXPECT().GetAnnotationValue(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(projectKey).AnyTimes()
	mitreStore.EXPECT().Get(gomock.Any()).Return(&storage.MitreAttackVector{}, nil).AnyTimes()
	defer mockCtrl.Finish()

	// Test with invalid password
	_, err := newJira(fakeJiraConfig, metadataGetter, mitreStore, cryptocodec.Singleton(), "stackrox")
	assert.Contains(t, err.Error(), "Status code: 401")

	// Test with valid username/password combo
	fakeJiraStorageConfig.Password = password
	_, err = newJira(fakeJiraConfig, metadataGetter, mitreStore, cryptocodec.Singleton(), "stackrox")
	require.NoError(t, err)

	// Test with valid bearer token
	fakeJiraStorageConfig.Password = token
	j, err := newJira(fakeJiraConfig, metadataGetter, mitreStore, cryptocodec.Singleton(), "stackrox")
	require.NoError(t, err)

	assert.Nil(t, j.Test(context.Background()))
	require.Len(t, fj.createdIssues, 1)
	issue := fj.createdIssues[0]
	assert.Equal(t, "StackRox Test Issue", issue.Fields.Description)
	assert.Equal(t, projectKey, issue.Fields.Project.Key)
	assert.Equal(t, "IssueWithPrio", issue.Fields.Type.Name)
	assert.Equal(t, "P3", issue.Fields.Priority.Name)

	testAlert := &storage.Alert{
		Id: "myAlertID",
		Policy: &storage.Policy{
			Id:             "myPolicyID",
			Name:           "myPolicy",
			Description:    "Fake policy",
			PolicySections: []*storage.PolicySection{},
			Severity:       storage.Severity_HIGH_SEVERITY,
		},
		Entity: &storage.Alert_Deployment_{Deployment: &storage.Alert_Deployment{
			Name: "myDeployment",
			Id:   "myDeploymentID",
		}},
		Time: types.TimestampNow(),
	}
	assert.NoError(t, j.AlertNotify(context.Background(), testAlert))
	require.Len(t, fj.createdIssues, 2)

	issue = fj.createdIssues[1]
	assert.Contains(t, issue.Fields.Description, "myDeployment")
	assert.Contains(t, issue.Fields.Description, "myDeploymentID")
	assert.Contains(t, issue.Fields.Description, "Fake policy")
	assert.Equal(t, "P1", issue.Fields.Priority.Name)
}
