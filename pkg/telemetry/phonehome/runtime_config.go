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
	// If the current key is DisabledKey, it won't be reconfigured.
	DisabledKey = "DISABLED"
	// TODO(ROX-17726): Remove hardcoded key.
	selfManagedKey = "eDd6QP8uWm0jCkAowEvijOPgeqtlulwR"
)

// RuntimeConfig defines some runtime features.
type RuntimeConfig struct {
	Key             string          `json:"storage_key_v1,omitempty"`
	APICallCampaign APICallCampaign `json:"api_call_campaign,omitempty"`
}

// getRuntimeConfig checks the provided defaultKey, and returns the adjusted
// runtime configuration, potentially downloaded from the cfgURL.
func getRuntimeConfig(cfgURL, defaultKey string) (*RuntimeConfig, error) {
	remoteCfg := &RuntimeConfig{
		Key: defaultKey,
	}
	if defaultKey == DisabledKey {
		return remoteCfg, nil
	}
	if cfgURL != "" {
		var err error
		remoteCfg, err = downloadConfig(cfgURL)
		if err != nil {
			return nil, err
		}
		if useRemoteKey(version.IsReleaseVersion(), remoteCfg, defaultKey) {
			log.Info("Telemetry configuration has been downloaded from ", cfgURL)
		} else {
			remoteCfg.Key = defaultKey
		}
	}
	return remoteCfg, remoteCfg.APICallCampaign.Compile()
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
	if err = json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return nil, errors.Wrap(err, "cannot decode telemetry configuration")
	}
	return cfg, cfg.APICallCampaign.Compile()
}

// useRemoteKey decides if the key from the downloaded configuration has to be
// used.
// We want to prevent accidental use of the production key, but still allow
// developers to test the functionality. So the current key has to match the one
// from the downloaded configuration for development installations.
// See unit tests for the examples.
func useRemoteKey(isRelease bool, remote *RuntimeConfig, localKey string) bool {
	if remote == nil {
		return false
	}
	if !isRelease {
		return remote.Key == localKey
	}
	return true
}
