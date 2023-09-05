package logging

import "github.com/stackrox/rox/pkg/centralevents"

// options for the logger.
type options struct {
	CentralEventsConverter centralevents.LogConverter
	CentralEventsStream    centralevents.Stream
}

// OptionsFunc allows setting log options for a logger.
type OptionsFunc = func(option *options)

// EnableCentralEvents enables the logger to send log statements of
// Errorw and Warnw as Central events to the end-user.
//
// Before enabling logging for your package, ensure that:
// * your module resolves to a specific domain (see pkg/centralevents/domain.go).
// * Central events emitted from your specific package have hints defined to help
//   users (see pkg/centralevents/hints.go).
func EnableCentralEvents() OptionsFunc {
	return func(option *options) {
		option.CentralEventsConverter = &zapLogConverter{}
		option.CentralEventsStream = centralevents.Singleton()
	}
}
