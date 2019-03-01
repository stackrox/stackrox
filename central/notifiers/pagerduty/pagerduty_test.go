package pagerduty

import (
	"os"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	apiKeyEnv = "PAGERDUTY_APIKEY"
)

func skip(t *testing.T) (apiKey string) {
	apiKey = os.Getenv(apiKeyEnv)
	if apiKey == "" {
		t.Skipf("Skipping PagerDuty integration test because %v is not defined", apiKey)
	}
	return
}

func getPagerDuty(t *testing.T) *pagerDuty {
	apiKey := skip(t)

	notifier := &storage.Notifier{
		UiEndpoint: "https://www.stackrox.com",
		Config: &storage.Notifier_Pagerduty{
			Pagerduty: &storage.PagerDuty{
				ApiKey: apiKey,
			},
		},
	}

	s, err := newPagerDuty(notifier)
	require.NoError(t, err)
	return s
}

func TestPagerDutyAlertNotify(t *testing.T) {
	p := getPagerDuty(t)
	assert.NoError(t, p.AlertNotify(fixtures.GetAlert()))
}

func TestPagerDutyNetworkPolicyYAMLNotify(t *testing.T) {
	s := getPagerDuty(t)

	assert.NoError(t, s.NetworkPolicyYAMLNotify(fixtures.GetYAML(), "test-cluster"))
}

func TestPagerDutyTest(t *testing.T) {
	s := getPagerDuty(t)
	assert.NoError(t, s.Test())
}

func TestPagerDutyAckAlert(t *testing.T) {
	p := getPagerDuty(t)
	alert := fixtures.GetAlert()
	alert.State = storage.ViolationState_SNOOZED
	assert.NoError(t, p.AckAlert(alert))
}

func TestPagerDutyResolveAlert(t *testing.T) {
	p := getPagerDuty(t)
	alert := fixtures.GetAlert()
	alert.State = storage.ViolationState_RESOLVED
	assert.NoError(t, p.ResolveAlert(alert))
}
