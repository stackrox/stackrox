//go:build integration

package splunk

import (
	"context"
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
