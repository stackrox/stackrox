package data

// TelemetryData is the top-level data structure that determines the shape of the telemetry data sent from central to
// the ingestion server.
type TelemetryData struct {
	Central  *CentralInfo   `json:"central,omitempty"`
	Clusters []*ClusterInfo `json:"clusters,omitempty"`
}
