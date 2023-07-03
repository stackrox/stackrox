package phonehome

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/httputil/proxy"
)

const (
	// DisabledKey is a key value which disables the telemetry collection.
	DisabledKey = "DISABLED"
	// TODO(ROX-17726): Remove hardcoded key.
	selfManagedKey = "eDd6QP8uWm0jCkAowEvijOPgeqtlulwR"
)

type remoteConfig struct {
	Key string `json:"storage_key_v1,omitempty"`
}

// DownloadConfig downloads the configuration from the provided url.
func DownloadConfig(u string) (*remoteConfig, error) {
	if u == "hardcoded" {
		// TODO(ROX-17726): Use the hardcoded key for now.
		return &remoteConfig{Key: selfManagedKey}, nil
	}
	client := http.Client{
		Timeout:   5 * time.Second,
		Transport: proxy.RoundTripper(),
	}
	resp, err := client.Get(u)
	if err != nil {
		return nil, errors.Wrap(err, "cannot download telemetry configuration")
	}
	var cfg *remoteConfig
	err = json.NewDecoder(resp.Body).Decode(&cfg)
	return cfg, errors.Wrap(err, "cannot decode telemetry configuration")
}

// ToDownload decides if a configuration with the key need to be downloaded.
// We want to prevent accidental use of the production key, but still allow
// developers to test the functionality. So download will only happen for
// development installations if both a key and an URL are provided. For release
// versions the key may be empty.
// See unit tests for the examples.
func ToDownload(isRelease bool, key, cfgURL string) bool {
	if cfgURL == "" {
		return false
	}
	if !isRelease {
		// Development versions must provide a key on top of the URL.
		return key != ""
	}
	return true
}

// UseRemoteKey decides if the key from the downloaded configuration has to be
// used.
// We want to prevent accidental use of the production key, but still allow
// developers to test the functionality. So the key from the environment
// has to match the one from the downloaded configuration for development
// installations.
// See unit tests for the examples.
func UseRemoteKey(isRelease bool, cfg *remoteConfig, localKey string) bool {
	if cfg == nil {
		return false
	}
	if !isRelease {
		// The key from the environment has to match for development versions.
		return cfg.Key == localKey
	}
	return true
}
