package events

import (
	"github.com/stackrox/rox/pkg/administration/events/stream"
	"github.com/stackrox/rox/pkg/logging"
)

// EnableAdministrationEvents configures the logger for the module to convert selected log statements
// to administration events.
func EnableAdministrationEvents() logging.OptionsFunc {
	return logging.EnableAdministrationEvents(stream.Singleton())
}
