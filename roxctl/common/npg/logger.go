package npg

import (
	"github.com/stackrox/rox/roxctl/common/logger"
)

// Logger wraps the common logging interface in a NP-Guard compatible logger.
// It mutes all warnings and errors as they are being returned explicitly by the library.
type Logger struct {
	l logger.Logger
}

// NewLogger returns new instance of log
func NewLogger(l logger.Logger) *Logger {
	return &Logger{
		l: l,
	}
}

// Debugf empty func, mutes debug messages as they are being returned explicitly by the library.
func (nl *Logger) Debugf(_ string, _ ...interface{}) {}

// Infof prints a formatted string with a newline, prefixed with INFO and colorized
func (nl *Logger) Infof(format string, o ...interface{}) {
	nl.l.InfofLn(format, o...)
}

// Warnf empty func, mutes the warnings as they are being returned explicitly by NP-Guard library
func (nl *Logger) Warnf(_ string, _ ...interface{}) {}

// Errorf empty func, mutes the errors as they are being returned explicitly by NP-Guard library
func (nl *Logger) Errorf(_ error, _ string, _ ...interface{}) {}
