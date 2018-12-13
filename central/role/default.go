package role

import (
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
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
var defaultRoles = []*storage.Role{
	permissions.NewRoleWithPermissions(None),
	permissions.NewRoleWithGlobalAccess(Admin, storage.Access_READ_WRITE_ACCESS),
	permissions.NewRoleWithGlobalAccess(Analyst, storage.Access_READ_ACCESS),
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
var DefaultRolesByName map[string]*storage.Role

func init() {
	DefaultRolesByName = make(map[string]*storage.Role)
	for _, role := range defaultRoles {
		DefaultRolesByName[role.GetName()] = role
	}
}
