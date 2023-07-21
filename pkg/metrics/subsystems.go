package metrics

// Subsystem represents a subsystem sent to Prometheus metrics.
type Subsystem string

// These consts enumerate all the subsystems that expose Prometheus metrics.
const (
	CentralSubsystem    Subsystem = "central"
	SensorSubsystem     Subsystem = "sensor"
	ComplianceSubsystem Subsystem = "compliance"
	ScannerSubsystem    Subsystem = "scanner"
)

func (s Subsystem) String() string {
	return string(s)
}
