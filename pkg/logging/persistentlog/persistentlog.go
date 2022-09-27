package persistentlog

import (
	"context"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/persistentlog"
)

// Reader provides functionality to read, parse and send persistent log to Postgres.
type Reader interface {
	// StartReader will start the persistent log reader process which will continuously read and send events until stopped.
	// Returns true if the reader can be started (log exists and can be read). Log file missing is not considered an error.
	StartReader(ctx context.Context) (bool, error)
	// StopReader will stop the reader if it's started. Will return false if it was already stopped.
	StopReader() bool
}

// NewReader returns a new instance of Reader
func NewReader() Reader {
	return &persistentLogReaderImpl{
		logPath:            logging.PostgresPersistentLoggingPath,
		stopC:              concurrency.NewSignal(),
		persistentLogStore: persistentlog.Singleton(),
	}
}
