package notifications

import "github.com/stackrox/rox/generated/storage"

// LogConverter converts a log entry to a storage.Notification
type LogConverter interface {
	Convert(msg string, context ...interface{}) *storage.Notification
}
