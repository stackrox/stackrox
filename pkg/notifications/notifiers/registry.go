package notifiers

import (
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/notifications/types"
)

// Creator is a function stub for a function that creates a Notifier
type Creator func(notifier *v1.Notifier) (types.Notifier, error)

// Registry is the map of plugin names to their creation functions
var Registry = map[string]Creator{}

// Add registers a plugin with their creator function
func Add(name string, creator Creator) {
	Registry[name] = creator
}
