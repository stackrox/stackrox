// build +integration

package jira

import (
	"os"
	"testing"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testJiraPassword = "JIRA_PASSWORD"
const testJiraUser = "JIRA_EMAIL"

func skip(t *testing.T) (string, string) {
	user := os.Getenv(testJiraUser)
	if user == "" {
		t.Skipf("Skipping jira integration test because %v is not defined", testJiraUser)
	}
	password := os.Getenv(testJiraPassword)
	if password == "" {
		t.Skipf("Skipping jira integration test because %v is not defined", testJiraPassword)
	}
	return user, password
}

func getJira(t *testing.T) *jira {
	user, password := skip(t)
	notifier := &v1.Notifier{
		UiEndpoint: "http://google.com",
		Config: &v1.Notifier_Jira{
			Jira: &v1.Jira{
				Username:  user,
				Password:  password,
				IssueType: "Bug",
				Url:       "https://stack-rox.atlassian.net/",
			},
		},
		LabelDefault: "AJIT",
	}

	j, err := newJira(notifier)
	require.NoError(t, err)
	return j
}

func TestJiraNotify(t *testing.T) {
	j := getJira(t)
	assert.NoError(t, j.AlertNotify(fixtures.GetAlert()))
}

func TestJiraTest(t *testing.T) {
	j := getJira(t)
	assert.NoError(t, j.Test())
}

func TestJiraBenchmarkNotify(t *testing.T) {
	j := getJira(t)
	schedule := &v1.BenchmarkSchedule{
		BenchmarkName: "CIS Docker Benchmark",
	}
	assert.NoError(t, j.BenchmarkNotify(schedule))
}
