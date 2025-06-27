package centralclient

import (
	"encoding/json"
	"time"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

func (cfg *centralConfig) appendRuntimeCampaign(campaign phonehome.APICallCampaign) {
	cfg.campaignMux.Lock()
	defer cfg.campaignMux.Unlock()
	cfg.telemetryCampaign = append(permanentTelemetryCampaign, campaign...)
	jc, _ := json.Marshal(cfg.telemetryCampaign)
	log.Info("API Telemetry campaign: ", string(jc))
}

// Reload fetches and applies the remote configuration. It will not enable an
// explicitely disabled configuraiton.
func (cfg *centralConfig) Reload() error {
	if !cfg.IsActive() {
		return nil
	}
	runtimeCfg, err := cfg.Reconfigure(
		env.TelemetryConfigURL.Setting(),
		env.TelemetryStorageKey.Setting(),
	)
	if err != nil {
		log.Errorf("Failed to reconfigure telemetry: %v.", err)
		return err
	}
	cfg.appendRuntimeCampaign(runtimeCfg.APICallCampaign)
	return nil
}

// StartPeriodicReload starts a goroutine that periodically fetches and reloads
// the remote configuration.
func (cfg *centralConfig) StartPeriodicReload(period time.Duration) {
	if url := env.TelemetryConfigURL.Setting(); url == "" || url == env.TelemetrySelfManagedURL {
		return
	}
	go func() {
		for range time.NewTicker(period).C {
			_ = cfg.Reload()
		}
	}()
}
