// +build integration

package splunk

import (
	"os"
	"testing"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	endpoint = "https://localhost:8088/services/collector/event"
	token    = "292A25D6-FF99-448C-BD90-7029FBD537BC"

	tokenEnv    = "SPLUNK_TOKEN"
	endpointEnv = "SPLUNK_ENDPOINT"
)

func skip(t *testing.T) (token string, endpoint string) {
	token = os.Getenv(tokenEnv)
	if token == "" {
		t.Skipf("Skipping splunk integration test because %v is not defined", tokenEnv)
	}
	endpoint = os.Getenv(endpointEnv)
	if endpoint == "" {
		t.Skipf("Skipping splunk integration test because %v is not defined", endpointEnv)
	}
	return
}

func getSplunk(t *testing.T) *splunk {
	token, endpoint := skip(t)

	notifier := &v1.Notifier{
		UiEndpoint: "http://google.com",
		Config: &v1.Notifier_Splunk{
			Splunk: &v1.Splunk{
				Token:        token,
				HttpEndpoint: endpoint,
			},
		},
	}

	s, err := newSplunk(notifier)
	require.NoError(t, err)
	return s
}

func TestSplunkAlertNotify(t *testing.T) {
	s := getSplunk(t)
	assert.NoError(t, s.AlertNotify(fixtures.GetAlert()))
}

func TestSplunkNetworkPolicyYAMLNotify(t *testing.T) {
	s := getSplunk(t)

	assert.NoError(t, s.NetworkPolicyYAMLNotify(fixtures.GetYAML(), "test-cluster"))
}

func TestSplunkTest(t *testing.T) {
	s := getSplunk(t)
	assert.NoError(t, s.Test())
}

func TestSplunkBenchmarkNotify(t *testing.T) {
	s := getSplunk(t)
	schedule := &v1.BenchmarkSchedule{
		BenchmarkName: "CIS Docker Benchmark",
	}
	assert.NoError(t, s.BenchmarkNotify(schedule))
}
