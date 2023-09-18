package centralsensor

// SensorCapability identifies a capability exposed by sensor.
type SensorCapability string

// String returns the string form of sensor capability.
func (s SensorCapability) String() string {
	return string(s)
}

// CentralCapability identifies a capability exposed by Central.
type CentralCapability string

func (s CentralCapability) String() string {
	return string(s)
}
