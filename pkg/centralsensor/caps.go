package centralsensor

// CentralCapability identifies a capability exposed by Central.
type CentralCapability string

// String returns the string form of central capability.
func (c CentralCapability) String() string {
	return string(c)
}

// SensorCapability identifies a capability exposed by sensor.
type SensorCapability string

// String returns the string form of sensor capability.
func (s SensorCapability) String() string {
	return string(s)
}
