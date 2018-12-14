// build +integration

package slack

import (
	"os"
	"testing"

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

func TestSlackAlertNotify(t *testing.T) {
	webhook := skip(t)
	s := slack{
		Notifier: &storage.Notifier{
			UiEndpoint:   "http://google.com",
			LabelDefault: webhook,
		},
	}
	assert.NoError(t, s.AlertNotify(fixtures.GetAlert()))
}

func TestSlackNetworkPolicyYAMLNotify(t *testing.T) {
	webhook := skip(t)
	s := slack{
		Notifier: &storage.Notifier{
			UiEndpoint:   "http://google.com",
			LabelDefault: webhook,
		},
	}

	assert.NoError(t, s.NetworkPolicyYAMLNotify(fixtures.GetYAML(), "test-cluster"))
}

func TestSlackTest(t *testing.T) {
	webhook := skip(t)
	s := slack{
		Notifier: &storage.Notifier{
			UiEndpoint:   "http://google.com",
			LabelDefault: webhook,
		},
	}
	assert.NoError(t, s.Test())
}

func TestSlackBenchmarkNotify(t *testing.T) {
	webhook := skip(t)
	s := slack{
		Notifier: &storage.Notifier{
			UiEndpoint:   "http://google.com",
			LabelDefault: webhook,
		},
	}
	schedule := &storage.BenchmarkSchedule{
		BenchmarkName: "CIS Docker Benchmark",
	}
	assert.NoError(t, s.BenchmarkNotify(schedule))
}
