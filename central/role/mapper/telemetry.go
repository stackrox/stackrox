package mapper

import (
	phonehome "github.com/stackrox/rox/central/telemetry/centralclient"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
)

// addUserToTenantGroup adds the given user to the central tenant group so that
// such users could be segmented by tenant properties.
func addUserToTenantGroup(user *storage.User) {
	c := phonehome.Singleton()
	// User ID is anonymized for privacy reasons.
	anonymizedUserID := c.HashUserID(user.GetId(), user.GetAuthProviderId())
	// The status of the telemetry client (enabled/disabled) should be known
	// at this point, so no goroutine, as there is no risk of blocking, and we
	// want to register the user as soon as possible.
	c.Group(append(c.WithGroups(), telemeter.WithUserID(anonymizedUserID))...)
}
