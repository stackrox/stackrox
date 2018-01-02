// build +integration

package slack

import (
	"os"
	"testing"

	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
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

func TestSlackNotify(t *testing.T) {
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

	alert := &v1.Alert{
		Id: "Alert1",
		Policy: &v1.Policy{
			Name:        "Vulnerable Container",
			Description: "Alert if the container contains vulnerabilities",
			Severity:    v1.Severity_LOW_SEVERITY,
		},
		Deployment: &v1.Deployment{
			Name: "nginx_server",
			Id:   "s79mdvmb6dsl",
			Containers: []*v1.Container{
				{
					Image: &v1.Image{
						Sha:      "SHA",
						Registry: "docker.io",
						Remote:   "library/nginx",
						Tag:      "latest",
					},
				},
			},
		},
		Violations: []*v1.Alert_Violation{
			{
				Message: "Deployment is affected by 'CVE-2017-15804'",
			},
			{
				"Deployment is affected by 'CVE-2017-15670'",
			},
		},
	}
	assert.NoError(t, s.Notify(alert))
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
