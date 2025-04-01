package centralclient

import (
	"context"
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

// getRuntimeConfig returns the runtime configuration with the configured key.
// If returns nil or error, telemetry should be disabled.
func getRuntimeConfig() (*phonehome.RuntimeConfig, error) {
	if env.OfflineModeEnv.BooleanSetting() {
		return nil, nil
	}
	runtimeCfg, err := phonehome.GetRuntimeConfig(
		env.TelemetryConfigURL.Setting(),
		env.TelemetryStorageKey.Setting(),
	)
	return runtimeCfg, errors.WithMessage(err, "failed to fetch runtime telemetry config")
}

func appendRuntimeCampaign(runtimeCfg *phonehome.RuntimeConfig) {
	campaignMux.Lock()
	defer campaignMux.Unlock()
	telemetryCampaign = permanentTelemetryCampaign
	if err := runtimeCfg.APICallCampaign.Compile(); err != nil {
		log.Errorf("Failed to initialize runtime telemetry campaign: %v.", err)
	} else {
		telemetryCampaign = append(telemetryCampaign, runtimeCfg.APICallCampaign...)
	}
	jc, _ := json.Marshal(telemetryCampaign)
	log.Info("API Telemetry campaign: ", string(jc))
}

func applyConfig() (bool, error) {
	runtimeCfg, err := getRuntimeConfig()
	if err != nil {
		return false, err
	}
	if runtimeCfg == nil {
		return false, nil
	}
	if err := getInstanceId(); err != nil {
		return false, err
	}
	applyRemoteConfig(runtimeCfg)
	return true, nil
}

func applyRemoteConfig(runtimeCfg *phonehome.RuntimeConfig) {
	startMux.Lock()
	defer startMux.Unlock()
	appendRuntimeCampaign(runtimeCfg)
	if config == nil {
		var props map[string]any
		config, props = getInstanceConfig(runtimeCfg.Key)
		config.Gatherer().AddGatherer(func(ctx context.Context) (map[string]any, error) {
			return props, nil
		})
	} else if config.StorageKey != runtimeCfg.Key {
		config.StorageKey = runtimeCfg.Key
		log.Info("New telemetry storage key: ", config.StorageKey)
	}
}

// Reload fetches and applies the remote configuration. It will not enable an
// explicitely disabled configuraiton.
func Reload() error {
	if enable, err := applyConfig(); err != nil {
		log.Errorf("Failed to reconfigure telemetry: %v.", err)
		return err
	} else if enable && enabled {
		Enable()
	} else {
		Disable()
	}
	return nil
}

// StartPeriodicReload starts a goroutine that periodically fetches and reloads
// the remote configuration.
func StartPeriodicReload(period time.Duration) {
	if url := env.TelemetryConfigURL.Setting(); url == "" || url == "hardcoded" {
		return
	}
	go func() {
		for range time.NewTicker(period).C {
			_ = Reload()
		}
	}()
}
