// build +integration

package slack

import (
	"os"
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/fixtures"
	"github.com/stretchr/testify/assert"
)

const testWebhookEnv = "SLACK_WEBHOOK"
const testChannelEnv = "SLACK_CHANNEL"

func skip(t *testing.T) (string, string) {
	webhook := os.Getenv(testWebhookEnv)
	if webhook == "" {
		t.Skipf("Skipping slack integration test because %v is not defined", testWebhookEnv)
	}
	channel := os.Getenv(testChannelEnv)
	if channel == "" {
		t.Skipf("Skipping slack integration test because %v is not defined", testChannelEnv)
	}
	return webhook, channel
}

func TestSlackAlertNotify(t *testing.T) {
	webhook, channel := skip(t)
	s := slack{
		config: config{
			Webhook: webhook,
			Channel: channel,
		},
		Notifier: &v1.Notifier{
			UiEndpoint: "http://google.com",
		},
	}
	assert.NoError(t, s.AlertNotify(fixtures.GetAlert()))
}

func TestSlackTest(t *testing.T) {
	webhook, channel := skip(t)
	s := slack{
		config: config{
			Webhook: webhook,
			Channel: channel,
		},
		Notifier: &v1.Notifier{
			UiEndpoint: "http://google.com",
		},
	}
	assert.NoError(t, s.Test())
}

func TestSlackBenchmarkNotify(t *testing.T) {
	webhook, channel := skip(t)
	s := slack{
		config: config{
			Webhook: webhook,
			Channel: channel,
		},
		Notifier: &v1.Notifier{
			UiEndpoint: "http://google.com",
		},
	}
	schedule := &v1.BenchmarkSchedule{
		Name: "CIS Docker Benchmark",
	}
	assert.NoError(t, s.BenchmarkNotify(schedule))
}
