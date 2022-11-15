package segment

import (
	"time"

	analytics "github.com/segmentio/analytics-go"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/telemetry/marketing"
	"github.com/stackrox/rox/pkg/version"
	"go.uber.org/zap/zapcore"
)

var (
	log      = logging.LoggerForModule()
	once     sync.Once
	instance *segmentTelemeter
)

// Enabled tells whether telemetry data collection is enabled.
func Enabled() bool {
	return env.SegmentAPIKey.Setting() != ""
}

type segmentTelemeter struct {
	client         analytics.Client
	deviceID       string
	staticIdentity map[string]any
}

// Ensure Telemeter interface implementation.
var _ = marketing.Telemeter((*segmentTelemeter)(nil))

func (t *segmentTelemeter) Identify(props map[string]any) {
	traits := analytics.NewTraits()
	identity := analytics.Identify{
		UserId: t.deviceID,
		Traits: traits,
	}
	if t.staticIdentity != nil {
		for k, v := range t.staticIdentity {
			traits.Set(k, v)
		}
		// Set the static properties only once:
		t.staticIdentity = nil
	}
	for k, v := range props {
		traits.Set(k, v)
	}
	log.Info("Identifying with ", identity)
	if err := t.client.Enqueue(identity); err != nil {
		log.Error("Cannot enqueue Segment identity event: %v", err)
	}
}

// Init creates and initializes a Segment telemeter instance.
func Init(config *marketing.Config) marketing.Telemeter {
	once.Do(func() {
		key := env.SegmentAPIKey.Setting()
		server := ""
		instance = initSegment(config, key, server)
	})
	return instance
}

type logWrapper struct {
	internal *logging.Logger
}

func (l *logWrapper) Logf(format string, args ...any) {
	l.internal.Logf(zapcore.InfoLevel, format, args)
}

func (l *logWrapper) Errorf(format string, args ...any) {
	l.internal.Errorf(format, args)
}

func initSegment(config *marketing.Config, key, server string) *segmentTelemeter {
	segmentConfig := analytics.Config{
		Endpoint: server,
		Interval: 1 * time.Hour,
		Logger:   &logWrapper{internal: log},
	}

	client, err := analytics.NewWithConfig(key, segmentConfig)
	if err != nil {
		log.Error("Cannot initialize Segment client: %v", err)
		return nil
	}

	return &segmentTelemeter{
		client:   client,
		deviceID: config.ID,
		staticIdentity: map[string]any{
			"Central version":    version.GetMainVersion(),
			"Chart version":      version.GetChartVersion(),
			"Orchestrator":       config.Orchestrator,
			"Kubernetes version": config.Version,
		},
	}
}

func (t *segmentTelemeter) Start() {
}

func (t *segmentTelemeter) Stop() {
	if t != nil {
		if err := t.client.Close(); err != nil {
			log.Error("Cannot close Segment client: %v", err)
		}
	}
}

func (t *segmentTelemeter) TrackProps(event string, props map[string]any) {
	if t == nil {
		return
	}
	log.Info("Tracking event ", event, " with ", props)

	if err := t.client.Enqueue(analytics.Track{
		UserId:     t.deviceID,
		Event:      event,
		Properties: props,
	}); err != nil {
		log.Error("Cannot enqueue Segment track event: %v", err)
	}
}

func (t *segmentTelemeter) TrackProp(event string, key string, value any) {
	t.TrackProps(event, map[string]any{key: value})
}

func (t *segmentTelemeter) Track(event string) {
	t.TrackProps(event, nil)
}
