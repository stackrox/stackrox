package logging

import (
	"github.com/stackrox/rox/pkg/administration/events"
	"github.com/stackrox/rox/pkg/features"
	"go.uber.org/zap/zapcore"
)

// options for the logger.
type options struct {
	AdministrationEventsConverter events.LogConverter
	AdministrationEventsStream    events.Stream
}

// OptionsFunc allows setting log options for a logger.
type OptionsFunc = func(option *options)

// EnableAdministrationEvents enables the logger to send log statements of
// Errorw and Warnw as administration events to the end-user.
//
// Before enabling logging for your package, ensure that:
//   - your module resolves to a specific domain (see pkg/administration/events/domain.go).
//   - Administration events emitted from your specific package have hints defined to help
//     users (see pkg/administration/events/hints.go).
func EnableAdministrationEvents(stream events.Stream) OptionsFunc {
	return func(option *options) {
		if features.AdministrationEvents.Enabled() {
			option.AdministrationEventsConverter = &zapLogConverter{
				consoleEncoder: zapcore.NewConsoleEncoder(config.EncoderConfig),
			}
			option.AdministrationEventsStream = stream
		}
	}
}
