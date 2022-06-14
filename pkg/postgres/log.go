package postgres

import (
	"runtime"

	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

// LogCallerOnPostgres logs the caller of this function when Postgres is enabled
// This helps trace where calls into deprecated features are coming from.
// e.g. calls into RocksDB and BoltDB initialization
func LogCallerOnPostgres(name string) {
	if features.PostgresDatastore.Enabled() {
		_, file, no, ok := runtime.Caller(2)
		if ok {
			log.Infof("%s called from %s#%d", name, file, no)
		}
	}
}
