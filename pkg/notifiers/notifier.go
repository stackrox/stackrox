package notifiers

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Notifier is the base notifier that all types of notifiers must implement
//
//go:generate mockgen-wrapper
type Notifier interface {
	// Close closes a notifier instances and releases all its resources.
	Close(context.Context) error
	// ProtoNotifier gets the proto version of the notifier
	ProtoNotifier() *storage.Notifier
	// Test sends a test message
	Test(context.Context) *NotifierError
}
