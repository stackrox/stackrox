package events

import "github.com/stackrox/rox/generated/storage"

// LogConverter converts a log entry to a storage.Event
type LogConverter interface {
	Convert(msg string, context ...interface{}) *storage.Event
}
