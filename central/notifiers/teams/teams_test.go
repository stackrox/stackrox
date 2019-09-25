package teams

import (
	"os"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
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

func TestTeamsAlertNotify(t *testing.T) {
	webhook := skip(t)
	s := teams{
		Notifier: &storage.Notifier{
			UiEndpoint:   "http://google.com",
			LabelDefault: webhook,
		},
	}
	assert.NoError(t, s.AlertNotify(fixtures.GetAlert()))
}

func TestTeamsRandomAlertNotify(t *testing.T) {
	webhook := skip(t)
	s := teams{
		Notifier: &storage.Notifier{
			UiEndpoint:   "http://google.com",
			LabelDefault: webhook,
		},
	}

	alert := fixtures.GetAlert()
	alert.Policy.Rationale = ""
	alert.Policy.Remediation = ""
	alert.Policy.Fields.AddCapabilities = []string{}
	alert.Policy.Fields.DropCapabilities = []string{}
	alert.Policy.Fields.Env = &storage.KeyValuePolicy{}
	alert.Policy.Fields.VolumePolicy = &storage.VolumePolicy{}
	alert.Policy.Fields.ImageName = &storage.ImageNamePolicy{}
	assert.NoError(t, s.AlertNotify(alert))

	alert.Policy = &storage.Policy{}
	assert.NoError(t, s.AlertNotify(alert))

	alert.Id = ""
	alert.Violations = []*storage.Alert_Violation{}
	alert.Deployment.ClusterId = ""
	alert.Deployment.ClusterName = ""
	assert.NoError(t, s.AlertNotify(alert))

	alert.Deployment = &storage.Alert_Deployment{}
	assert.NoError(t, s.AlertNotify(alert))

	alert = &storage.Alert{}
	assert.NoError(t, s.AlertNotify(alert))
}

func TestTeamsNetworkPolicyYAMLNotify(t *testing.T) {
	webhook := skip(t)
	s := teams{
		Notifier: &storage.Notifier{
			UiEndpoint:   "http://google.com",
			LabelDefault: webhook,
		},
	}

	assert.NoError(t, s.NetworkPolicyYAMLNotify(fixtures.GetYAML(), "test-cluster"))
}

func TestTeamsTest(t *testing.T) {
	webhook := skip(t)
	s := teams{
		Notifier: &storage.Notifier{
			UiEndpoint:   "http://google.com",
			LabelDefault: webhook,
		},
	}
	assert.NoError(t, s.Test())
}
