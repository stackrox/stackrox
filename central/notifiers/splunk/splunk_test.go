//go:build integration

package splunk

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cryptoutils/cryptocodec"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
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

	notifier := &storage.Notifier{
		UiEndpoint: "http://google.com",
		Config: &storage.Notifier_Splunk{
			Splunk: &storage.Splunk{
				HttpToken:    token,
				HttpEndpoint: endpoint,
			},
		},
	}

	s, err := newSplunk(notifier, cryptocodec.Singleton(), "stackrox")
	require.NoError(t, err)
	return s
}

func TestSplunkAlertNotify(t *testing.T) {
	s := getSplunk(t)
	assert.NoError(t, s.AlertNotify(context.Background(), fixtures.GetAlert()))
}

func TestSplunkTest(t *testing.T) {
	s := getSplunk(t)
	assert.NoError(t, s.Test(context.Background()))
}

func TestSendEvent(t *testing.T) {
	fakeSpunkSvc := &fakeSplunk{
		tb:              t,
		expectedPayload: expectedMarshaledEvent,
	}
	server := httptest.NewServer(fakeSpunkSvc)
	defer server.Close()

	baseNotifier := &storage.Notifier{
		UiEndpoint: server.URL,
		Config: &storage.Notifier_Splunk{
			Splunk: &storage.Splunk{
				HttpToken:    "abcdefgh",
				HttpEndpoint: server.URL,
				SourceTypes:  defaultSourceTypeMap,
			},
		},
	}

	splunkNotifier, err := newSplunk(baseNotifier, cryptocodec.Singleton(), "stackrox")
	require.NoError(t, err)

	err = splunkNotifier.sendEvent(context.Background(), testAlert, alertSourceTypeKey)
	assert.NoError(t, err)
}

var (
	testAlert = fixtures.GetSerializationTestAlert()

	expectedMarshaledEvent = `{
	"source": "stackrox",
	"sourcetype": "stackrox-alert",
	"event": {
		"@type": "storage.Alert",
		"id": "aeaaaaaa-bbbb-4011-0000-111111111111",
		"clusterId": "caaaaaaa-bbbb-4011-0000-111111111111",
		"clusterName": "prod cluster",
		"namespace": "stackrox",
		"processViolation": {
			"message": "This is a process violation"
		},
		"violations": [
			{
				"message": "Deployment is affected by 'CVE-2017-15670'"
			},
			{
				"message": "This is a kube event violation",
				"keyValueAttrs": {
					"attrs": [
						{"key": "pod", "value": "nginx"},
						{"key": "container", "value": "nginx"}
					]
				}
			}
		]
	}
}`
)

type fakeSplunk struct {
	tb              testing.TB
	expectedPayload string
}

func (s *fakeSplunk) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusBadRequest)
		s.tb.Error("Bad HTTP method", r.Method)
		return
	}

	body := r.Body
	defer func() { _ = body.Close() }()
	bodyData, err := io.ReadAll(body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		s.tb.Error("Error reading body", err)
		return
	}

	match := assert.JSONEq(s.tb, s.expectedPayload, string(bodyData))
	if !match {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/text")
	w.WriteHeader(200)
	_, err = w.Write([]byte("ok"))
	assert.NoError(s.tb, err)
}
