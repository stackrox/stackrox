package netpol

import (
	"github.com/stackrox/rox/roxctl/common/logger"
)

// npgLogger wraps the common logging interface in a NP-Guard compatible logger.
// It mutes all warnings and errors as they are being returned explicitly by the library.
type npgLogger struct {
	l logger.Logger
}

func newNpgLogger(l logger.Logger) *npgLogger {
	return &npgLogger{
		l: l,
	}
}

func (nl *npgLogger) Debugf(format string, o ...interface{}) {}
func (nl *npgLogger) Infof(format string, o ...interface{}) {
	nl.l.InfofLn(format, o...)
}
func (nl *npgLogger) Warnf(format string, o ...interface{})             {}
func (nl *npgLogger) Errorf(err error, format string, o ...interface{}) {}
