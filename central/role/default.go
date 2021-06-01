package role

import (
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/set"
)

// All builtin, immutable role names are declared in the block below.
const (
	// Admin is a role that's, well, authorized to do anything.
	Admin = "Admin"

	// Analyst is a role that has read access to all resources
	Analyst = "Analyst"

	// None role has no access.
	None = user.NoneRole

	// ContinuousIntegration is for CI piplines.
	ContinuousIntegration = "Continuous Integration"

	// SensorCreator is a role that has the minimal privileges required to create a sensor.
	SensorCreator = "Sensor Creator"
)

// DefaultRoleNames is a string set containing the names of all default (built-in) Roles.
var DefaultRoleNames = set.NewStringSet(Admin, Analyst, None, ContinuousIntegration, SensorCreator)
