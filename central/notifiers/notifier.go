package notifiers

import (
	"github.com/stackrox/rox/generated/storage"
)

// Notifier is the base notifier that all types of notifiers must implement
//go:generate mockgen-wrapper
type Notifier interface {
	// ProtoNotifier gets the proto version of the notifier
	ProtoNotifier() *storage.Notifier
	// Test sends a test message
	Test() error
}
