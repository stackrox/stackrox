package centralevents

import (
	"github.com/stackrox/rox/generated/storage"
)

// LogConverter converts a log entry to a storage.CentralEvent.
type LogConverter interface {
	Convert(msg string, level string, module string, context ...interface{}) *storage.CentralEvent
}
