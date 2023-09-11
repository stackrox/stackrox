package logging

import "github.com/stackrox/rox/pkg/administration/events"

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
func EnableAdministrationEvents() OptionsFunc {
	return func(option *options) {
		option.AdministrationEventsConverter = &zapLogConverter{}
		option.AdministrationEventsStream = events.Singleton()
	}
}
