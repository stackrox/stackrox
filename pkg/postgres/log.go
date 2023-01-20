package postgres

import (
	"runtime"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/utils"
)

var log = logging.LoggerForModule()

// LogCallerOnPostgres logs the caller of this function when Postgres is enabled
// This helps trace where calls into deprecated features are coming from.
// e.g. calls into RocksDB and BoltDB initialization
func LogCallerOnPostgres(name string) {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		_, file, no, ok := runtime.Caller(2)
		if ok {
			log.Infof("%s called from %s#%d", name, file, no)
			utils.Should(errors.New("Unexpected access call"))
		}
	}
}
