package utils

import (
	"runtime"

	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

// LogCaller logs the caller of this function
// This helps trace where calls into deprecated features are coming from.
// e.g. calls into RocksDB and BoltDB initialization
func LogCaller(name string) {
	_, file, no, ok := runtime.Caller(2)
	if ok {
		log.Infof("%s called from %s#%d", name, file, no)
	}
}
