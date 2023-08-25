// build +integration

package jira

import (
	"context"
	"os"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	mitreMocks "github.com/stackrox/rox/pkg/mitre/datastore/mocks"
	notifierMocks "github.com/stackrox/rox/pkg/notifiers/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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

func getJira(t *testing.T) (*jira, *gomock.Controller) {
	mockCtrl := gomock.NewController(t)
	metadataGetter := notifierMocks.NewMockMetadataGetter(mockCtrl)
	metadataGetter.EXPECT().GetAnnotationValue(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("").AnyTimes()
	mitreStore := mitreMocks.NewMockAttackReadOnlyDataStore(mockCtrl)
	mitreStore.EXPECT().Get(gomock.Any()).Return(&storage.MitreAttackVector{}, nil).AnyTimes()

	user, password := skip(t)
	notifier := &storage.Notifier{
		UiEndpoint: "http://google.com",
		Config: &storage.Notifier_Jira{
			Jira: &storage.Jira{
				Username:  user,
				Password:  password,
				IssueType: "Bug",
				Url:       "https://stack-rox.atlassian.net/",
			},
		},
		LabelDefault: "AJIT",
	}

	j, err := NewJira(notifier, metadataGetter, mitreStore)
	require.NoError(t, err)
	return j, mockCtrl
}

func TestJiraAlertNotify(t *testing.T) {
	j, mockCtrl := getJira(t)
	defer mockCtrl.Finish()

	assert.NoError(t, j.AlertNotify(context.Background(), fixtures.GetAlert()))
}

func TestJiraNetworkPolicyYAMLNotify(t *testing.T) {
	j, mockCtrl := getJira(t)
	defer mockCtrl.Finish()

	assert.NoError(t, j.NetworkPolicyYAMLNotify(context.Background(), fixtures.GetYAML(), "test-cluster"))
}

func TestJiraTest(t *testing.T) {
	j, mockCtrl := getJira(t)
	defer mockCtrl.Finish()

	assert.NoError(t, j.Test(context.Background()))
}
