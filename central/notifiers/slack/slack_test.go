// build +integration

package slack

import (
	"context"
	"encoding/json"
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

const testWebhookEnv = "SLACK_WEBHOOK"

func skip(t *testing.T) string {
	webhook := os.Getenv(testWebhookEnv)
	if webhook == "" {
		t.Skipf("Skipping slack integration test because %v is not defined", testWebhookEnv)
	}
	return webhook
}

func getSlackWithMock(t *testing.T, notifier *storage.Notifier) (*slack, *gomock.Controller) {
	mockCtrl := gomock.NewController(t)
	mitreStore := mitreMocks.NewMockAttackReadOnlyDataStore(mockCtrl)
	mitreStore.EXPECT().Get(gomock.Any()).Return(&storage.MitreAttackVector{}, nil).AnyTimes()
	metadataGetter := notifierMocks.NewMockMetadataGetter(mockCtrl)
	metadataGetter.EXPECT().GetAnnotationValue(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("").AnyTimes()
	s, err := NewSlack(notifier, metadataGetter, mitreStore)
	assert.NoError(t, err)

	return s, mockCtrl
}

func TestSlackAlertNotify(t *testing.T) {
	webhook := skip(t)
	s, mockCtrl := getSlackWithMock(t, &storage.Notifier{
		UiEndpoint:   "http://google.com",
		LabelDefault: webhook,
	})
	defer mockCtrl.Finish()

	assert.NoError(t, s.AlertNotify(context.Background(), fixtures.GetAlert()))
}

func TestSlackNetworkPolicyYAMLNotify(t *testing.T) {
	webhook := skip(t)
	s, mockCtrl := getSlackWithMock(t, &storage.Notifier{
		UiEndpoint:   "http://google.com",
		LabelDefault: webhook,
	})
	defer mockCtrl.Finish()

	assert.NoError(t, s.NetworkPolicyYAMLNotify(context.Background(), fixtures.GetYAML(), "test-cluster"))
}

func TestSlackTest(t *testing.T) {
	webhook := skip(t)
	s, mockCtrl := getSlackWithMock(t, &storage.Notifier{
		UiEndpoint:   "http://google.com",
		LabelDefault: webhook,
	})
	defer mockCtrl.Finish()

	assert.NoError(t, s.Test(context.Background()))
}

func TestNotificationContainsFlatFields(t *testing.T) {
	alert := fixtures.GetAlert()
	s, mockCtrl := getSlackWithMock(t, &storage.Notifier{
		UiEndpoint:   "https://stackrox.example.com",
		LabelDefault: "https://hooks.slack.com/test",
	})
	defer mockCtrl.Finish()

	body, err := s.getDescription(alert)
	require.NoError(t, err)

	meta := getEntityMetadata(alert)
	n := notification{
		Attachments: []attachment{{Text: body}},
		Summary:     "test summary",
		AlertID:     alert.GetId(),
		Severity:    "High",
		Policy:      alert.GetPolicy().GetName(),
		Cluster:     meta.cluster,
		Namespace:   meta.namespace,
		Entity:      meta.name,
		EntityType:  meta.entityType,
		Violations:  violationMessages(alert),
		Description: body,
	}

	data, err := json.Marshal(&n)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))

	assert.Equal(t, alert.GetId(), raw["alert_id"])
	assert.Equal(t, "High", raw["severity"])
	assert.Equal(t, alert.GetPolicy().GetName(), raw["policy"])
	assert.Equal(t, "prod cluster", raw["cluster"])
	assert.Equal(t, "stackrox", raw["namespace"])
	assert.Equal(t, "nginx_server", raw["entity"])
	assert.Equal(t, "Deployment", raw["entity_type"])
	assert.Contains(t, raw["violations"], "CVE-2017-15804")
	assert.Contains(t, raw["violations"], "This is a process violation")
	assert.NotEmpty(t, raw["description"])
	assert.NotNil(t, raw["attachments"], "attachments should still be present for legacy webhook compatibility")
}

func TestGetEntityMetadata(t *testing.T) {
	cases := map[string]struct {
		alert              *storage.Alert
		expectedName       string
		expectedEntityType string
		expectedCluster    string
		expectedNamespace  string
	}{
		"deployment": {
			alert:              fixtures.GetAlert(),
			expectedName:       "nginx_server",
			expectedEntityType: "Deployment",
			expectedCluster:    "prod cluster",
			expectedNamespace:  "stackrox",
		},
		"resource": {
			alert:              fixtures.GetResourceAlert(),
			expectedName:       "my-secret",
			expectedEntityType: "Resource",
			expectedCluster:    "prod cluster",
			expectedNamespace:  "stackrox",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			meta := getEntityMetadata(tc.alert)
			assert.Equal(t, tc.expectedName, meta.name)
			assert.Equal(t, tc.expectedEntityType, meta.entityType)
			assert.Equal(t, tc.expectedCluster, meta.cluster)
			assert.Equal(t, tc.expectedNamespace, meta.namespace)
		})
	}
}

func TestViolationMessages(t *testing.T) {
	alert := fixtures.GetAlert()
	msgs := violationMessages(alert)
	assert.Contains(t, msgs, "CVE-2017-15804")
	assert.Contains(t, msgs, "CVE-2017-15670")
	assert.Contains(t, msgs, "This is a process violation")
}
