package run

import "time"

// Config holds in-process local-sensor startup options shared by the CLI and benchmarks.
type Config struct {
	Duration          time.Duration
	CentralEndpoint   string
	FakeWorkloadFile  string
	PoliciesFile      string
	RecordK8s         bool
	RecordK8sFile     string
	ReplayK8s         bool
	ReplayK8sFile     string
	Delay             time.Duration
	Verbose           bool
	MetricsEnabled    bool
	MetricsPort       string
	SkipCentralOutput bool
	CentralOutput     string
	OutputFormat      string
	NoCPUProfile      bool
	NoMemProfile      bool
	PprofServer       bool
	FakeCollector     bool
	Namespace         string
	OperatorInstall   bool
}
