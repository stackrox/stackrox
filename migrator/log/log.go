package log

import (
	"github.com/stackrox/stackrox/pkg/logging"
)

var (
	logger = logging.ModuleForName("Migrator").Logger()
)

// WriteToStderr is a helper function to write to stderr.
func WriteToStderr(s string) {
	logger.Info(s)
}

// WriteToStderrf writes to stderr with a format string.
func WriteToStderrf(format string, args ...interface{}) {
	logger.Infof(format, args...)
}
