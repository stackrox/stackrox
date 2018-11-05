package role

import (
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
)

// All currently-valid role names are declared in the block below.
const (
	// Admin is a role that's, well, authorized to do anything.
	Admin = "Admin"

	// ContinuousIntegration is for CI piplines.
	ContinuousIntegration = "Continuous Integration"

	// SensorCreator is a role that has the minimal privileges required to create a sensor.
	SensorCreator = "Sensor Creator"
)

// DefaultRoles holds the... default roles.
var DefaultRoles = map[string]*v1.Role{
	Admin: permissions.NewAllAccessRole(Admin),
	ContinuousIntegration: permissions.NewRoleWithPermissions(ContinuousIntegration,
		permissions.View(resources.Detection),
	),
	SensorCreator: permissions.NewRoleWithPermissions(SensorCreator,
		permissions.View(resources.Cluster),
		permissions.Modify(resources.Cluster),
		permissions.View(resources.ServiceIdentity),
	),
}
