package utils

import (
	"github.com/stackrox/rox/pkg/uuid"
)

// GetMaskedDeploymentID returns a deterministic ID value different
// from the input ID in order to hide deployment IDs for deployments
// out of the requester access scope.
func GetMaskedDeploymentID(id string, name string) string {
	return uuid.NewV5FromNonUUIDs(id, name).String()
}
