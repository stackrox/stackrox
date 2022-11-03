package amplitude

import (
	"sync"
	"time"

	"github.com/amplitude/analytics-go/amplitude"
	"github.com/amplitude/analytics-go/amplitude/types"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/telemetry/marketing"
	"github.com/stackrox/rox/pkg/version"
)

var (
	log      = logging.LoggerForModule()
	once     sync.Once
	instance *amplitudeTelemeter
)

type amplitudeTelemeter struct {
	client   amplitude.Client
	opts     *types.EventOptions
	identity amplitude.Identify
}

// Ensure Telemeter interface implementation.
var _ = marketing.Telemeter((*amplitudeTelemeter)(nil))

func (t *amplitudeTelemeter) Identify(props map[string]any) {
	t.identity = amplitude.Identify{}
	for k, v := range props {
		t.identity.Set(k, v)
	}
	t.client.Identify(t.identity, amplitude.EventOptions{UserID: t.opts.DeviceID, DeviceID: t.opts.DeviceID})
}

// Init creates and initializes an amplitude telemeter instance.
func Init(config *marketing.Config) marketing.Telemeter {
	once.Do(func() {
		key := env.AmplitudeAPIKey.Setting()
		server := ""
		instance = initAmplitude(config, key, server)
	})
	return instance
}

func initAmplitude(config *marketing.Config, key, server string) *amplitudeTelemeter {
	amplitudeConfig := amplitude.NewConfig(key)
	if server != "" {
		amplitudeConfig.ServerURL = server
	}
	amplitudeConfig.FlushInterval = 1 * time.Hour
	amplitudeConfig.Logger = log

	client := amplitude.NewClient(amplitudeConfig)

	identify := amplitude.Identify{}
	identify.SetOnce("Central version", version.GetMainVersion())
	identify.SetOnce("Chart version", version.GetChartVersion())

	return &amplitudeTelemeter{
		client:   client,
		identity: identify,
		opts: &amplitude.EventOptions{
			DeviceID:  config.ID,
			ProductID: version.GetMainVersion(),
			Platform:  config.Version,
		},
	}
}

func (t *amplitudeTelemeter) Start() {
}

func (t *amplitudeTelemeter) Stop() {
	if t != nil {
		t.client.Flush()
		t.client.Shutdown()
	}
}

func (t *amplitudeTelemeter) TrackProps(userAgent, event string, props map[string]any) {
	if t == nil {
		return
	}
	opts := *t.opts
	opts.AppVersion = userAgent
	t.client.Track(amplitude.Event{
		UserID:          t.opts.DeviceID,
		DeviceID:        t.opts.DeviceID,
		EventType:       event,
		EventProperties: props,
		EventOptions:    opts,
	})
}

func (t *amplitudeTelemeter) TrackProp(userAgent, event string, key string, value any) {
	t.TrackProps(userAgent, event, map[string]any{key: value})
}

func (t *amplitudeTelemeter) Track(userAgent, event string) {
	t.TrackProps(userAgent, event, nil)
}
