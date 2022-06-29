package loghelper

import "github.com/stackrox/rox/migrator/log"

// LogWrapper redirects all log messages to standard error for migration
type LogWrapper struct{}

// Debugf is a helper function to write debug message to stderr
func (l *LogWrapper) Debugf(format string, args ...interface{}) {
	log.WriteToStderrf(format, args...)
}

// WriteToStderr is a helper function to write to stderr.
func (l *LogWrapper) WriteToStderr(s string) {
	log.WriteToStderr(s)
}

// WriteToStderrf writes to stderr with a format string.
func (l *LogWrapper) WriteToStderrf(format string, args ...interface{}) {
	log.WriteToStderrf(format, args...)
}
