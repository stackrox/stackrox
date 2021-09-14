// build +integration

package slack

import (
	"context"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	mitreMocks "github.com/stackrox/rox/central/mitre/datastore/mocks"
	namespaceMocks "github.com/stackrox/rox/central/namespace/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/assert"
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
	nsStore := namespaceMocks.NewMockDataStore(mockCtrl)
	mitreStore := mitreMocks.NewMockMitreAttackReadOnlyDataStore(mockCtrl)
	nsStore.EXPECT().SearchNamespaces(gomock.Any(), gomock.Any()).Return([]*storage.NamespaceMetadata{}, nil).AnyTimes()
	mitreStore.EXPECT().Get(gomock.Any()).Return(&storage.MitreAttackVector{}, nil).AnyTimes()
	s, err := newSlack(notifier, nsStore, mitreStore)
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
