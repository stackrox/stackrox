package mapper

import (
	phonehome "github.com/stackrox/rox/central/telemetry/centralclient"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
)

// addUserToTenantGroup adds the given user to the central tenant group so that
// such users could be segmented by tenant properties.
func addUserToTenantGroup(user *storage.User) {
	if c := phonehome.Singleton(); c.IsEnabled() {
		c.Telemeter().Group(nil, telemeter.WithUserID(c.HashUserID(user.GetId(), user.GetAuthProviderId())),
			telemeter.WithGroups(c.GroupType, c.GroupID))
	}
}
