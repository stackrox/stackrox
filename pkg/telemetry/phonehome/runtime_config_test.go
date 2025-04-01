package phonehome

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
)

func Test_toDownload(t *testing.T) {
	tests := map[string]struct {
		release bool
		key     string
		cfgURL  string

		download bool
	}{
		"a": {release: false, key: "", cfgURL: "", download: false},
		"b": {release: false, key: "abc", cfgURL: "", download: false},
		"c": {release: false, key: "", cfgURL: "url", download: false},
		"d": {release: false, key: "abc", cfgURL: "url", download: true},

		"e": {release: true, key: "", cfgURL: "", download: false},
		"f": {release: true, key: "abc", cfgURL: "", download: false},
		"g": {release: true, key: "", cfgURL: "url", download: true},
		"h": {release: true, key: "abc", cfgURL: "url", download: false},

		"j": {release: false, key: "", cfgURL: env.TelemetrySelfManagedURL, download: false},
		"k": {release: false, key: "abc", cfgURL: env.TelemetrySelfManagedURL, download: true},
		"l": {release: true, key: "", cfgURL: env.TelemetrySelfManagedURL, download: true},
		"m": {release: true, key: "abc", cfgURL: env.TelemetrySelfManagedURL, download: false},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := toDownload(tt.release, tt.key, tt.cfgURL); got != tt.download {
				t.Errorf("toDownload() = %v, want %v", got, tt.download)
			}
		})
	}
}

func Test_useRemoteKey(t *testing.T) {
	tests := map[string]struct {
		release  bool
		cfg      *RuntimeConfig
		localKey string

		useRemote bool
	}{
		"a": {release: false, cfg: nil, localKey: "", useRemote: false},
		"b": {release: false, cfg: &RuntimeConfig{Key: "abc"}, localKey: "", useRemote: false},
		"c": {release: false, cfg: nil, localKey: "", useRemote: false},
		"d": {release: false, cfg: &RuntimeConfig{Key: "abc"}, localKey: "abc", useRemote: true},
		"e": {release: false, cfg: &RuntimeConfig{Key: "abc"}, localKey: "def", useRemote: false},

		"f": {release: true, cfg: nil, localKey: "", useRemote: false},
		"g": {release: true, cfg: &RuntimeConfig{Key: "abc"}, localKey: "", useRemote: true},
		"h": {release: true, cfg: nil, localKey: "", useRemote: false},
		"i": {release: true, cfg: &RuntimeConfig{Key: "abc"}, localKey: "abc", useRemote: true},
		"j": {release: true, cfg: &RuntimeConfig{Key: "abc"}, localKey: "def", useRemote: true},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := useRemoteKey(tt.release, tt.cfg, tt.localKey); got != tt.useRemote {
				t.Errorf("useRemoteKey() = %v, want %v", got, tt.useRemote)
			}
		})
	}
}

func Test_download(t *testing.T) {
	cfg, err := downloadConfig(env.TelemetrySelfManagedURL)
	assert.NoError(t, err)
	assert.NotEmpty(t, cfg.Key)

	const remoteKey = "remotekey"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"storage_key_v1": "` + remoteKey + `" }`))
	}))
	defer server.Close()

	cfg, err = downloadConfig(server.URL)
	assert.NoError(t, err)
	assert.Equal(t, remoteKey, cfg.Key)
}

func Test_GetKey(t *testing.T) {
	const devVersion = "4.4.1-dev"
	const remoteKey = "remotekey"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"storage_key_v1": "` + remoteKey + `" }`))
	}))
	defer server.Close()

	// There's no way to test a release version in a test binary.
	testutils.SetMainVersion(t, devVersion)

	tests := map[string]struct {
		defaultKey string
		cfgURL     string

		expectedKey string
		expectedErr error
	}{
		"a": {defaultKey: "", cfgURL: ""},
		"b": {defaultKey: "", cfgURL: env.TelemetrySelfManagedURL},
		"c": {defaultKey: DisabledKey, cfgURL: ""},
		"d": {defaultKey: "abc", cfgURL: env.TelemetrySelfManagedURL,
			expectedKey: "abc",
		},
		"e": {defaultKey: "ignored", cfgURL: ":bad url:",
			expectedErr: errors.New("missing protocol scheme"),
		},
		"f": {defaultKey: selfManagedKey, cfgURL: env.TelemetrySelfManagedURL,
			expectedKey: selfManagedKey,
		},
		"g": {defaultKey: remoteKey, cfgURL: server.URL,
			expectedKey: remoteKey,
		},
		"h": {defaultKey: "whatever", cfgURL: server.URL,
			expectedKey: "whatever",
		},
		"i": {defaultKey: "whatever",
			expectedKey: "whatever",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := GetRuntimeConfig(tt.cfgURL, tt.defaultKey)
			if tt.expectedErr != nil {
				assert.ErrorContains(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}
			if tt.expectedKey == "" {
				assert.Nil(t, got, name)
			} else {
				assert.Equal(t, tt.expectedKey, got.Key, name)
			}
		})
	}
}
