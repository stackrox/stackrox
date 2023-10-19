package pagerduty

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cryptoutils/cryptocodec"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	//#nosec G101 -- This is a false positive
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

	s, err := newPagerDuty(notifier, cryptocodec.Singleton(), "stackrox")
	require.NoError(t, err)
	return s
}

func TestPagerDutyAlertNotify(t *testing.T) {
	p := getPagerDuty(t)
	assert.NoError(t, p.AlertNotify(context.Background(), fixtures.GetAlert()))
}

func TestPagerDutyTest(t *testing.T) {
	s := getPagerDuty(t)
	assert.NoError(t, s.Test(context.Background()))
}

func TestPagerDutyAckAlert(t *testing.T) {
	p := getPagerDuty(t)
	alert := fixtures.GetAlert()
	alert.State = storage.ViolationState_SNOOZED
	assert.NoError(t, p.AckAlert(context.Background(), alert))
}

func TestPagerDutyResolveAlert(t *testing.T) {
	p := getPagerDuty(t)
	alert := fixtures.GetAlert()
	alert.State = storage.ViolationState_RESOLVED
	assert.NoError(t, p.ResolveAlert(context.Background(), alert))
}

func TestMarshalingAlert(t *testing.T) {
	cases := []struct {
		name  string
		alert *storage.Alert
	}{
		{"regular alert", fixtures.GetAlert()},
		{"image alert", fixtures.GetImageAlert()},
		{"resource alert", fixtures.GetResourceAlert()},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			alert := (*marshalableAlert)(c.alert)

			data, err := json.Marshal(alert)
			require.NoError(t, err)
			require.NotNil(t, data)

			var unmarshaledAlert *marshalableAlert
			require.NoError(t, json.Unmarshal(data, &unmarshaledAlert))

			require.True(t, reflect.DeepEqual(alert, unmarshaledAlert))
		})
	}
}
