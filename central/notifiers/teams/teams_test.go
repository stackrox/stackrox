package teams

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	namespaceMocks "github.com/stackrox/stackrox/central/namespace/datastore/mocks"
	"github.com/stackrox/stackrox/central/notifiers"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/fixtures"
	"github.com/stretchr/testify/assert"
)

const testWebhookEnv = "TEAMS_WEBHOOK"

func skip(t *testing.T) string {
	webhook := os.Getenv(testWebhookEnv)
	if webhook == "" {
		t.Skipf("Skipping teams integration test because %v is not defined", testWebhookEnv)
	}
	return webhook
}

func getTeamsWithMock(t *testing.T, notifier *storage.Notifier) (*teams, *gomock.Controller) {
	mockCtrl := gomock.NewController(t)
	nsStore := namespaceMocks.NewMockDataStore(mockCtrl)
	nsStore.EXPECT().SearchNamespaces(gomock.Any(), gomock.Any()).Return([]*storage.NamespaceMetadata{}, nil).AnyTimes()

	s, err := newTeams(notifier, nsStore)
	assert.NoError(t, err)

	return s, mockCtrl
}

func TestTeamsAlertNotify(t *testing.T) {
	webhook := skip(t)
	s, mockCtrl := getTeamsWithMock(t, &storage.Notifier{
		UiEndpoint:   "http://google.com",
		LabelDefault: webhook,
	})
	defer mockCtrl.Finish()

	assert.NoError(t, s.AlertNotify(context.Background(), fixtures.GetAlert()))
}

func TestTeamsRandomAlertNotify(t *testing.T) {
	webhook := skip(t)
	s, mockCtrl := getTeamsWithMock(t, &storage.Notifier{
		UiEndpoint:   "http://google.com",
		LabelDefault: webhook,
	})
	defer mockCtrl.Finish()

	alert := fixtures.GetAlert()
	alert.Policy.Rationale = ""
	alert.Policy.Remediation = ""
	assert.NoError(t, s.AlertNotify(context.Background(), alert))

	alert.Policy = &storage.Policy{}
	assert.NoError(t, s.AlertNotify(context.Background(), alert))

	alert.Id = ""
	alert.Violations = []*storage.Alert_Violation{}
	alert.GetDeployment().ClusterId = ""
	alert.GetDeployment().ClusterName = ""
	assert.NoError(t, s.AlertNotify(context.Background(), alert))

	alert.Entity = &storage.Alert_Deployment_{
		Deployment: &storage.Alert_Deployment{},
	}
	assert.NoError(t, s.AlertNotify(context.Background(), alert))

	alert = &storage.Alert{}
	assert.NoError(t, s.AlertNotify(context.Background(), alert))
}

func TestTeamsNetworkPolicyYAMLNotify(t *testing.T) {
	webhook := skip(t)
	s, mockCtrl := getTeamsWithMock(t, &storage.Notifier{
		UiEndpoint:   "http://google.com",
		LabelDefault: webhook,
	})
	defer mockCtrl.Finish()

	assert.NoError(t, s.NetworkPolicyYAMLNotify(context.Background(), fixtures.GetYAML(), "test-cluster"))
}

func TestTeamsTest(t *testing.T) {
	webhook := skip(t)
	s, mockCtrl := getTeamsWithMock(t, &storage.Notifier{
		UiEndpoint:   "http://google.com",
		LabelDefault: webhook,
	})
	defer mockCtrl.Finish()

	assert.NoError(t, s.Test(context.Background()))
}

func TestPolicySeverityEnumConverter(t *testing.T) {
	for k := range storage.Severity_value {
		actual, err := notifiers.GetNotifiersCompatiblePolicySeverity(k)
		assert.Nil(t, err)
		prefix := strings.Split(k, "_")[0]
		expected := strings.Title(strings.ToLower(prefix))
		assert.Equal(t, actual, expected)
	}
}
