package centralclient

import (
	"encoding/json"
	"time"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

func (c *centralClient) appendRuntimeCampaign(campaign phonehome.APICallCampaign) {
	c.campaignMux.Lock()
	defer c.campaignMux.Unlock()
	c.telemetryCampaign = append(permanentTelemetryCampaign, campaign...)
	jc, err := json.Marshal(c.telemetryCampaign)
	if err != nil {
		log.Warnw("Failed to marshal the API Telemetry campaign to JSON", logging.Err(err))
	} else {
		log.Info("API Telemetry campaign: ", string(jc))
	}
}

// Reload fetches and applies the remote configuration. It will not enable an
// explicitely disabled configuraiton.
func (c *centralClient) Reload() error {
	if !c.IsActive() {
		return errox.InvalidArgs.New("Telemetry client is not configured")
	}
	runtimeCfg, err := c.Reconfigure(
		env.TelemetryConfigURL.Setting(),
		env.TelemetryStorageKey.Setting(),
	)
	if err != nil {
		log.Errorf("Failed to reconfigure telemetry: %v.", err)
		return err
	}
	c.appendRuntimeCampaign(runtimeCfg.APICallCampaign)
	return nil
}

// StartPeriodicReload starts a goroutine that periodically fetches and reloads
// the remote configuration.
func (c *centralClient) StartPeriodicReload(period time.Duration) {
	if url := env.TelemetryConfigURL.Setting(); url == "" || url == env.TelemetrySelfManagedURL {
		return
	}
	go func() {
		for range time.NewTicker(period).C {
			_ = c.Reload()
		}
	}()
}
