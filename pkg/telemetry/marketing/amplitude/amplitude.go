package amplitude

import (
	"time"

	"github.com/amplitude/analytics-go/amplitude"
	"github.com/amplitude/analytics-go/amplitude/types"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/telemetry/marketing"
	"github.com/stackrox/rox/pkg/version"
)

var (
	log      = logging.LoggerForModule()
	once     sync.Once
	instance *amplitudeTelemeter
)

type amplitudeTelemeter struct {
	client         amplitude.Client
	opts           *types.EventOptions
	staticIdentity map[string]any
}

// Ensure Telemeter interface implementation.
var _ = marketing.Telemeter((*amplitudeTelemeter)(nil))

func (t *amplitudeTelemeter) Identify(props map[string]any) {
	identity := amplitude.Identify{}
	if t.staticIdentity != nil {
		for k, v := range t.staticIdentity {
			identity.SetOnce(k, v)
		}
		// Set the static properties only once:
		t.staticIdentity = nil
	}
	for k, v := range props {
		identity.Set(k, v)
	}
	log.Info("Identifying with ", identity)

	t.client.Identify(identity, amplitude.EventOptions{UserID: t.opts.DeviceID, DeviceID: t.opts.DeviceID})
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

	return &amplitudeTelemeter{
		client: client,
		opts: &amplitude.EventOptions{
			DeviceID: config.ID,
		},
		staticIdentity: map[string]any{
			"Central version":    version.GetMainVersion(),
			"Chart version":      version.GetChartVersion(),
			"Orchestrator":       config.Orchestrator,
			"Kubernetes version": config.Version,
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

func (t *amplitudeTelemeter) TrackProps(event string, props map[string]any) {
	if t == nil {
		return
	}
	log.Info("Tracking event ", event, " with ", props)
	t.client.Track(amplitude.Event{
		UserID:          t.opts.DeviceID,
		DeviceID:        t.opts.DeviceID,
		EventType:       event,
		EventProperties: props,
		EventOptions:    *t.opts,
	})
}

func (t *amplitudeTelemeter) TrackProp(event string, key string, value any) {
	t.TrackProps(event, map[string]any{key: value})
}

func (t *amplitudeTelemeter) Track(event string) {
	t.TrackProps(event, nil)
}
