//go:build test

package phonehome

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
)

func Test_download(t *testing.T) {
	const devVersion = "4.4.1-dev"
	const remoteKey = "remotekey"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"storage_key_v1": "` + remoteKey + `" }`))
	}))
	defer server.Close()

	// This method is only available for binaries built with `test` flag, so
	// there is no way to unit test a real release binary without the test flag.
	testutils.SetMainVersion(t, devVersion)

	// For dev main version the runtime config will not return any key.
	// The actual data download is tested with Test_download()
	tests := map[string]struct {
		url         string
		expectedKey string
		expectedErr bool
	}{
		"empty":     {url: "", expectedErr: true},
		"hardcoded": {url: env.TelemetrySelfManagedURL, expectedKey: hardcodedSelfManagedKey},
		"localhost": {url: server.URL, expectedKey: remoteKey},
		"bad":       {url: ":bad url:", expectedErr: true},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rc, err := downloadConfig(tt.url)
			if tt.expectedErr {
				assert.Error(t, err)
				assert.Nil(t, rc)
			} else if assert.NotNil(t, rc) {
				assert.Equal(t, tt.expectedKey, rc.Key, name)
			}
		})
	}
}

func Test_getRuntimeConfig_Campaign(t *testing.T) {
	const remoteKey = "remotekey"
	testutils.SetMainVersion(t, "4.4.1-dev")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"storage_key_v1": "` + remoteKey + `",
			"api_call_campaign": [
				{"method": "{put,delete}"},
				{"headers": {"Accept-Encoding": "*json*"}}
			]
		}`))
	}))
	defer server.Close()

	cfg, err := downloadConfig(server.URL)
	assert.NoError(t, err)
	assert.Equal(t, &RuntimeConfig{
		Key: remoteKey,
		APICallCampaign: APICallCampaign{
			MethodPattern("{put,delete}"),
			HeaderPattern("Accept-Encoding", "*json*"),
		},
	}, cfg)
}
