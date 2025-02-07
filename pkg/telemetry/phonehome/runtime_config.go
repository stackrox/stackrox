package phonehome

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/version"
)

const (
	// DisabledKey is a key value which disables the telemetry collection.
	DisabledKey = "DISABLED"
	// TODO(ROX-17726): Remove hardcoded key.
	selfManagedKey = "eDd6QP8uWm0jCkAowEvijOPgeqtlulwR"
)

// RuntimeConfig defines some runtime features.
type RuntimeConfig struct {
	Key             string          `json:"storage_key_v1,omitempty"`
	APICallCampaign APICallCampaign `json:"api_call_campaign,omitempty"`
}

// GetRuntimeConfig checks the provided defaultKey, and returns the adjusted
// runtime configuration, potentially downloaded from the cfgURL, or nil value
// if telemetry has to be disabled.
func GetRuntimeConfig(cfgURL, defaultKey string) (*RuntimeConfig, error) {
	key := defaultKey
	if key == DisabledKey {
		return nil, nil
	}

	remoteCfg := &RuntimeConfig{
		Key: key,
	}
	if toDownload(version.IsReleaseVersion(), key, cfgURL) {
		var err error
		remoteCfg, err = downloadConfig(cfgURL)
		if err != nil {
			return nil, err
		}
		if useRemoteKey(version.IsReleaseVersion(), remoteCfg, key) {
			log.Info("Telemetry configuration has been downloaded from ", cfgURL)
		} else {
			remoteCfg.Key = key
		}
	}

	// The downloaded key can be empty or 'DISABLED', so check again here.
	if remoteCfg == nil || remoteCfg.Key == "" || remoteCfg.Key == DisabledKey {
		return nil, nil
	}
	return remoteCfg, nil
}

// downloadConfig downloads the configuration from the provided url.
func downloadConfig(u string) (*RuntimeConfig, error) {
	// TODO(ROX-17726): Remove this clause:
	if u == env.TelemetrySelfManagedURL {
		// Use the hardcoded key for now.
		return &RuntimeConfig{Key: selfManagedKey}, nil
	}
	client := http.Client{
		Timeout:   5 * time.Second,
		Transport: proxy.RoundTripper(),
	}
	resp, err := client.Get(u)
	if err != nil {
		return nil, errors.Wrap(err, "cannot download telemetry configuration")
	}
	var cfg *RuntimeConfig
	err = json.NewDecoder(resp.Body).Decode(&cfg)
	return cfg, errors.Wrap(err, "cannot decode telemetry configuration")
}

// toDownload decides if a configuration with the key need to be downloaded.
// We want to prevent accidental use of the production key, but still allow
// developers to test the functionality. So download will only happen for
// development installations if both a key and an URL are provided. For release
// versions the key should be empty.
// See unit tests for the examples.
func toDownload(isRelease bool, key, cfgURL string) bool {
	if cfgURL == "" {
		return false
	}
	if !isRelease {
		// Development versions must provide a key on top of the URL.
		return key != ""
	}
	return key == ""
}

// useRemoteKey decides if the key from the downloaded configuration has to be
// used.
// We want to prevent accidental use of the production key, but still allow
// developers to test the functionality. So the key from the environment
// has to match the one from the downloaded configuration for development
// installations.
// See unit tests for the examples.
func useRemoteKey(isRelease bool, cfg *RuntimeConfig, localKey string) bool {
	if cfg == nil {
		return false
	}
	if !isRelease {
		// The key from the environment has to match for development versions.
		return cfg.Key == localKey
	}
	return true
}
