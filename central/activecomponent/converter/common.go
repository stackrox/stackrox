package converter

import "github.com/stackrox/rox/generated/storage"

// CompleteActiveComponent includes explicit ComponentID and DeploymentID.
type CompleteActiveComponent struct {
	ActiveComponent *storage.ActiveComponent
	ComponentID     string
	DeploymentID    string
}
