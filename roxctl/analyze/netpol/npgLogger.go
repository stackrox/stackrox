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

func (nl *npgLogger) Debugf(_ string, _ ...interface{}) {}
func (nl *npgLogger) Infof(format string, o ...interface{}) {
	nl.l.InfofLn(format, o...)
}
func (nl *npgLogger) Warnf(_ string, _ ...interface{})           {}
func (nl *npgLogger) Errorf(_ error, _ string, _ ...interface{}) {}
