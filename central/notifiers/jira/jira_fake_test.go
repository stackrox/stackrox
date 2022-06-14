package jira

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	jiraLib "github.com/andygrunwald/go-jira"
	"github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	mitreMocks "github.com/stackrox/stackrox/central/mitre/datastore/mocks"
	namespaceMocks "github.com/stackrox/stackrox/central/namespace/datastore/mocks"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeJira is a fake JIRA backend that implements exactly the APIs that the JIRA notifier needs (and only to the extent
// required by the notifier code).
// This is in no way intended to be a realistic model of the JIRA API, it only allows us to exercise notifier code paths
// in this test.
type fakeJira struct {
	t                  *testing.T
	username, password string

	priorities []jiraLib.Priority
	project    jiraLib.MetaProject

	createdIssues []jiraLib.Issue
}

func (j *fakeJira) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/2/priority", j.handlePriority)
	mux.HandleFunc("/rest/api/2/issue/createmeta", j.handleCreateMeta)
	mux.HandleFunc("/rest/api/2/issue", j.handleCreateIssue)

	if j.username == "" && j.password == "" {
		return mux
	}

	expectedAuthHeader := fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", j.username, j.password))))
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Authorization") != expectedAuthHeader {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		mux.ServeHTTP(w, req)
	})
}

func (j *fakeJira) handlePriority(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	require.NoError(j.t, json.NewEncoder(w).Encode(j.priorities))
}

func (j *fakeJira) handleCreateMeta(w http.ResponseWriter, req *http.Request) {
	queryVals := req.URL.Query()
	expectedQueryVals := url.Values{
		"expand":      []string{"projects.issuetypes.fields"},
		"projectKeys": []string{j.project.Key},
	}
	if !assert.Equal(j.t, expectedQueryVals, queryVals) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	cmi := jiraLib.CreateMetaInfo{
		Projects: []*jiraLib.MetaProject{&j.project},
	}
	require.NoError(j.t, json.NewEncoder(w).Encode(&cmi))
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
	const (
		username = "fakejirauser"
		password = "fakejirapassword"

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
			},
			{
				Name: "IssueWithPrio",
				Fields: map[string]interface{}{
					"priority": true,
				},
			},
		},
	}

	fj := fakeJira{
		t:          t,
		username:   username,
		password:   password,
		priorities: priorities,
		project:    project,
	}

	testSrv := httptest.NewServer(fj.Handler())
	defer testSrv.Close()

	fakeJiraConfig := &storage.Notifier{
		Name:         "FakeJIRA",
		UiEndpoint:   "https://central.stackrox",
		Type:         "jira",
		LabelDefault: projectKey,
		Config: &storage.Notifier_Jira{
			Jira: &storage.Jira{
				Url:       testSrv.URL,
				Username:  "fakejirauser",
				Password:  "fakejirapassword",
				IssueType: "IssueWithPrio",
			},
		},
	}

	mockCtrl := gomock.NewController(t)
	nsStore := namespaceMocks.NewMockDataStore(mockCtrl)
	mitreStore := mitreMocks.NewMockMitreAttackReadOnlyDataStore(mockCtrl)
	mitreStore.EXPECT().Get(gomock.Any()).Return(&storage.MitreAttackVector{}, nil).AnyTimes()
	j, err := newJira(fakeJiraConfig, nsStore, mitreStore)
	defer mockCtrl.Finish()

	require.NoError(t, err)

	assert.NoError(t, j.Test(context.Background()))
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
