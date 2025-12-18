package debugactions

import (
	"github.com/stackrox/rox/central/debugactions/manager"
	"github.com/stackrox/rox/pkg/buildinfo"
)

// ExecuteRegisteredAction executes the action registered for the given identifier.
// If no action is registered, it will do nothing
func ExecuteRegisteredAction(identifier string) {
	if !buildinfo.ReleaseBuild {
		manager.Singleton().ExecRegisteredAction(identifier)
	}
}
