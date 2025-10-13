package option

import (
	"github.com/stackrox/rox/pkg/administration/events/stream"
	"github.com/stackrox/rox/pkg/logging"
)

// EnableAdministrationEvents enables the logger to create administration events for all structured log statements.
func EnableAdministrationEvents() logging.OptionsFunc {
	return logging.EnableAdministrationEvents(stream.Singleton())
}
