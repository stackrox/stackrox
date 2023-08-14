package events

import (
	"github.com/stackrox/rox/generated/storage"
)

// Writer is capable of writing events. The events may be written to an underlying storage.
type Writer interface {
	Write(msg *storage.Event)
}
