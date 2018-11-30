package role

import (
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
)

// All builtin, immutable role names are declared in the block below.
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

// DefaultRoles are the pre-defined roles available.
var defaultRoles = []*v1.Role{
	permissions.NewRoleWithPermissions(None),
	permissions.NewRoleWithGlobalAccess(Admin, v1.Access_READ_WRITE_ACCESS),
	permissions.NewRoleWithGlobalAccess(Analyst, v1.Access_READ_ACCESS),
	permissions.NewRoleWithPermissions(ContinuousIntegration,
		permissions.View(resources.Detection),
	),
	permissions.NewRoleWithPermissions(SensorCreator,
		permissions.View(resources.Cluster),
		permissions.Modify(resources.Cluster),
		permissions.View(resources.ServiceIdentity),
	),
}

// DefaultRolesByName holds the default roles mapped by name.
var DefaultRolesByName map[string]*v1.Role

func init() {
	DefaultRolesByName = make(map[string]*v1.Role)
	for _, role := range defaultRoles {
		DefaultRolesByName[role.GetName()] = role
	}
}
