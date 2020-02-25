package data

// RoxComponentInfo holds telemetry data for StackRox-specific deployments such as Central, Sensor, and Collector
type RoxComponentInfo struct {
	Version  string       `json:"version"`
	Process  *ProcessInfo `json:"process,omitempty"`
	Restarts int          `json:"restarts,omitempty"`
}
