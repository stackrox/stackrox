package retryablehttp

import (
	"github.com/stackrox/rox/pkg/logging"
)

// DebugLogger adapts logging.Logger to the retryablehttp.Logger interface.
type DebugLogger struct {
	logger logging.Logger
}

// Printf implements the retryablehttp.Logger interface by logging at Debug level.
func (l *DebugLogger) Printf(format string, v ...interface{}) {
	l.logger.Debugf(format, v...)
}

func NewDebugLogger(logger logging.Logger) *DebugLogger {
	return &DebugLogger{logger: logger}
}
