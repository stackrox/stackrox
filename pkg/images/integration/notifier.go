package integration

import (
	"github.com/stackrox/rox/generated/api/v1"
)

// ToNotify is the client view of a notifier, allowing it to execute notifications but not to modify
// who receives those notifications.
type ToNotify interface {
	NotifyUpdated(integration *v1.ImageIntegration) error
	NotifyRemoved(id string) error
}

// Notifier provides an interface for configuring and executing notifications of image integration changes.
type Notifier interface {
	NotifyUpdated(integration *v1.ImageIntegration) error
	NotifyRemoved(id string) error

	AddOnUpdate(func(*v1.ImageIntegration) error)
	AddOnRemove(func(id string) error)
}

// NewToNotify returns a new  ToNotify instance that updates the set when a notification is received.
func NewToNotify(is Set) ToNotify {
	// Notifications of updates or removals will update the set accordingly.
	notifier := &notifierImpl{}
	notifier.AddOnUpdate(is.UpdateImageIntegration)
	notifier.AddOnRemove(is.RemoveImageIntegration)

	return notifier
}
