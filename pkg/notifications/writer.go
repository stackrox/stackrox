package notifications

import (
	"github.com/stackrox/rox/generated/storage"
)

// Writer is capable of writing notifications. The notification may be written to an underlying storage.
type Writer interface {
	Write(msg *storage.Notification)
}
