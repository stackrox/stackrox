package role

// All currently-valid role names are declared in the block below.
const (
	// Admin is a role that's, well, authorized to do anything.
	Admin = "Admin"

	// ContinuousIntegration is for CI piplines.
	ContinuousIntegration = "ContinuousIntegration"

	// SensorCreator is a role that has the minimal privileges required to create a sensor.
	SensorCreator = "Sensor Creator"
)
