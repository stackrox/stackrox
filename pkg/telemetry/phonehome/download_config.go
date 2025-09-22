package phonehome

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/httputil/proxy"
)

// hardcodedSelfManagedKey is the key used by the self-managed installations.
const hardcodedSelfManagedKey = "eDd6QP8uWm0jCkAowEvijOPgeqtlulwR"

// RuntimeConfig defines some runtime features.
type RuntimeConfig struct {
	Key             string          `json:"storage_key_v1,omitempty"`
	APICallCampaign APICallCampaign `json:"api_call_campaign,omitempty"`
}

// downloadConfig downloads the configuration from the provided url.
func downloadConfig(url string) (*RuntimeConfig, error) {
	if url == env.TelemetrySelfManagedURL {
		return &RuntimeConfig{Key: hardcodedSelfManagedKey}, nil
	}

	client := http.Client{
		Timeout:   5 * time.Second,
		Transport: proxy.RoundTripper(),
	}
	resp, err := client.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "cannot download telemetry configuration")
	}
	var cfg *RuntimeConfig
	if err = json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return nil, errors.Wrap(err, "cannot decode telemetry configuration")
	}
	if err = cfg.APICallCampaign.Compile(); err != nil {
		return nil, errors.Wrap(err, "cannot compile API call campaign")
	}
	return cfg, nil
}
