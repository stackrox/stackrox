package npg

import (
	"github.com/stackrox/rox/roxctl/common/logger"
)

// log wraps the common logging interface in a NP-Guard compatible logger.
// It mutes all warnings and errors as they are being returned explicitly by the library.
type log struct {
	l logger.Logger
}

// NewLogger returns new instance of log
func NewLogger(l logger.Logger) *log {
	return &log{
		l: l,
	}
}

func (nl *log) Debugf(_ string, _ ...interface{}) {}
func (nl *log) Infof(format string, o ...interface{}) {
	nl.l.InfofLn(format, o...)
}
func (nl *log) Warnf(_ string, _ ...interface{})           {}
func (nl *log) Errorf(_ error, _ string, _ ...interface{}) {}
