package notifiers

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
)

// Creator is a function stub for a function that creates a Notifier
type Creator func(notifier *storage.Notifier) (Notifier, error)

// Registry is the map of plugin names to their creation functions
var Registry = map[string]Creator{}

// Add registers a plugin with their creator function
func Add(name string, creator Creator) {
	Registry[name] = creator
}

// CreateNotifier checks to make sure the integration exists and then tries to generate a new Notifier
// returns an error if the creation was unsuccessful
func CreateNotifier(notifier *storage.Notifier) (Notifier, error) {
	creator, exists := Registry[notifier.Type]
	if !exists {
		return nil, fmt.Errorf("Notifier with type '%v' does not exist", notifier.Type)
	}
	return creator(notifier)
}
