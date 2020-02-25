package data

// ProcessMemInfo contains telemetry data about the resources used by a process
type ProcessMemInfo struct {
	CurrentAllocBytes   int64 `json:"currentAllocBytes"`
	CurrentAllocObjects int64 `json:"currentAllocObjects"`

	TotalAllocBytes   int64 `json:"totalAllocBytes"`
	TotalAllocObjects int64 `json:"totalAllocObjects"`

	SysMemBytes int64 `json:"sysMemBytes"`

	NumGCs     int64   `json:"numGCs,omitempty"`
	GCFraction float64 `json:"gcFraction,omitempty"`
}

// ProcessInfo contains telemetry data about a process
type ProcessInfo struct {
	NumGoroutines int `json:"numGoroutines,omitempty"`
	NumCPUs       int `json:"numCPUs,omitempty"`

	Memory *ProcessMemInfo `json:"memory,omitempty"`
}
