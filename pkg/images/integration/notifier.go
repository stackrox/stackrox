package integration

import (
	"github.com/stackrox/rox/generated/storage"
)

// ToNotify is the client view of a notifier, allowing it to execute notifications but not to modify
// who receives those notifications.
type ToNotify interface {
	NotifyUpdated(integration *storage.ImageIntegration) error
	NotifyRemoved(id string) error
}

// NewToNotify returns a new  ToNotify instance that updates the set when a notification is received.
func NewToNotify(is Set) ToNotify {
	// Notifications of updates or removals will update the set accordingly.
	notifier := &notifierImpl{}
	notifier.addOnUpdate(is.UpdateImageIntegration)
	notifier.addOnRemove(is.RemoveImageIntegration)

	return notifier
}
