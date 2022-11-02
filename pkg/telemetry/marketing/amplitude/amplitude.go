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
	log  = logging.LoggerForModule()
	once sync.Once
	a    *ampl
)

type ampl struct {
	client   amplitude.Client
	opts     *types.EventOptions
	identity amplitude.Identify
}

// Ensure Telemeter interface implementation.
var _ = marketing.Telemeter((*ampl)(nil))

func (t *ampl) Identify(props map[string]any) {
	t.identity = amplitude.Identify{}
	for k, v := range props {
		t.identity.Set(k, v)
	}
	t.client.Identify(t.identity, amplitude.EventOptions{UserID: t.opts.DeviceID, DeviceID: t.opts.DeviceID})
}

func Init(device *marketing.Device) marketing.Telemeter {
	once.Do(func() {
		key := env.AmplitudeApiKey.Setting()
		server := ""
		a = initAmplitude(device, key, server)
	})
	return a
}

func initAmplitude(device *marketing.Device, key, server string) *ampl {
	amplitude_config := amplitude.NewConfig(key)
	if server != "" {
		amplitude_config.ServerURL = server
	}
	amplitude_config.FlushInterval = 1 * time.Hour
	amplitude_config.Logger = log

	log.Info("Telemetry device ID:", device.ID)

	client := amplitude.NewClient(amplitude_config)

	identify := amplitude.Identify{}
	identify.SetOnce("Central version", version.GetMainVersion())
	identify.SetOnce("Chart version", version.GetChartVersion())

	return &ampl{
		client:   client,
		identity: identify,
		opts: &amplitude.EventOptions{
			DeviceID:  device.ID,
			ProductID: version.GetMainVersion(),
			Platform:  device.Version,
		},
	}
}

func (t *ampl) Start() {
}

func (t *ampl) Stop() {
	if t != nil {
		t.client.Flush()
		t.client.Shutdown()
	}
}

func (t *ampl) TrackProps(userAgent, event string, props map[string]any) {
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

func (t *ampl) TrackProp(userAgent, event string, key string, value any) {
	t.TrackProps(userAgent, event, map[string]any{key: value})
}

func (t *ampl) Track(userAgent, event string) {
	t.TrackProps(userAgent, event, nil)
}
