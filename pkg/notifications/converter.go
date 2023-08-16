package notifications

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging/structured"
)

// LogConverter converts a log entry to a storage.Notification.
type LogConverter interface {
	Convert(msg string, module string, context ...interface{}) *storage.Notification
}

// DefaultLogConverter returns the default log converter to be used.
func DefaultLogConverter() LogConverter {
	return &structured.zapConverter{}
}
