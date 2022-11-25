package segment

import (
	"time"

	segment "github.com/segmentio/analytics-go"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"go.uber.org/zap/zapcore"
)

var (
	log = logging.LoggerForModule()
)

// Enabled tells whether telemetry data collection is enabled.
func Enabled() bool {
	return env.TelemetryStorageKey.Setting() != ""
}

type segmentTelemeter struct {
	client segment.Client
	config *phonehome.Config
}

// Ensure Telemeter interface implementation.
var _ = phonehome.Telemeter((*segmentTelemeter)(nil))

func (t *segmentTelemeter) Identify(props map[string]any) {
	traits := segment.NewTraits()
	identity := segment.Identify{
		UserId: t.config.ID,
		Traits: traits,
	}
	if t.config.Identity != nil {
		for k, v := range t.config.Identity {
			traits.Set(k, v)
		}
	}
	for k, v := range props {
		traits.Set(k, v)
	}
	log.Info("Identifying with ", identity)
	if err := t.client.Enqueue(identity); err != nil {
		log.Error("Cannot enqueue Segment identity event: ", err)
	}
}

// NewTelemeter creates and initializes a Segment telemeter instance.
func NewTelemeter(config *phonehome.Config) phonehome.Telemeter {
	key := env.TelemetryStorageKey.Setting()
	server := ""
	return initSegment(config, key, server)
}

type logWrapper struct {
	internal *logging.Logger
}

func (l *logWrapper) Logf(format string, args ...any) {
	l.internal.Logf(zapcore.InfoLevel, format, args...)
}

func (l *logWrapper) Errorf(format string, args ...any) {
	l.internal.Errorf(format, args...)
}

func initSegment(config *phonehome.Config, key, server string) *segmentTelemeter {
	segmentConfig := segment.Config{
		Endpoint: server,
		Interval: 5 * time.Minute,
		Logger:   &logWrapper{internal: log},
		DefaultContext: &segment.Context{
			Extra: map[string]any{
				"Central ID": config.ID,
				"Tenant ID":  config.TenantID,
			},
		},
	}

	client, err := segment.NewWithConfig(key, segmentConfig)
	if err != nil {
		log.Error("Cannot initialize Segment client: ", err)
		return nil
	}

	return &segmentTelemeter{
		client: client,
		config: config,
	}
}

func (t *segmentTelemeter) Start() {
}

func (t *segmentTelemeter) Stop() {
	if t != nil {
		if err := t.client.Close(); err != nil {
			log.Error("Cannot close Segment client: ", err)
		}
	}
}

func (t *segmentTelemeter) Track(event, userID string, props map[string]any) {
	if t == nil {
		return
	}

	if err := t.client.Enqueue(segment.Track{
		UserId:     userID,
		Event:      event,
		Properties: props,
	}); err != nil {
		log.Error("Cannot enqueue Segment track event: ", err)
	}
}
