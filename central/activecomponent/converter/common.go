package converter

import "github.com/stackrox/stackrox/generated/storage"

// CompleteActiveComponent includes explicit ComponentID and DeploymentID.
type CompleteActiveComponent struct {
	ActiveComponent *storage.ActiveComponent
	ComponentID     string
	DeploymentID    string
}
