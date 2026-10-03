package metrics

// Subsystem represents a subsystem sent to Prometheus metrics.
type Subsystem string

// These consts enumerate all the subsystems that expose Prometheus metrics.
const (
	CentralSubsystem          Subsystem = "central"
	CentralWorkerSubsystem    Subsystem = "central_worker"
	SensorSubsystem           Subsystem = "sensor"
	AdmissionControlSubsystem Subsystem = "admission_control"
	ComplianceSubsystem       Subsystem = "compliance"
	ScannerSubsystem          Subsystem = "scanner"
	BackgroundWorkerSubsystem Subsystem = "background_worker"
)

func (s Subsystem) String() string {
	return string(s)
}
