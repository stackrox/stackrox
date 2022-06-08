package loghelper

import "github.com/stackrox/rox/migrator/log"

type LogWrapper struct{}

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
