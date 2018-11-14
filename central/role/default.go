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

	// Analyst is a role that has read access to all resources
	Analyst = "Analyst"

	// None role has no access.
	None = "None"

	// ContinuousIntegration is for CI piplines.
	ContinuousIntegration = "Continuous Integration"

	// SensorCreator is a role that has the minimal privileges required to create a sensor.
	SensorCreator = "Sensor Creator"
)

// DefaultRoles holds the... default roles.
var DefaultRoles = map[string]*v1.Role{
	None:    permissions.NewRoleWithPermissions(None),
	Analyst: permissions.NewReadOnlyRole(Analyst),
	Admin:   permissions.NewReadWriteRole(Admin),
	ContinuousIntegration: permissions.NewRoleWithPermissions(ContinuousIntegration,
		permissions.View(resources.Detection),
	),
	SensorCreator: permissions.NewRoleWithPermissions(SensorCreator,
		permissions.View(resources.Cluster),
		permissions.Modify(resources.Cluster),
		permissions.View(resources.ServiceIdentity),
	),
}
