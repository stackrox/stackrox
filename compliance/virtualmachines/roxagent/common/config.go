package common

import "time"

type Config struct {
	DaemonMode            bool
	Debug                 bool
	IndexHostPath         string
	IndexInterval         time.Duration
	MaxInitialReportDelay time.Duration
	RepoToCPEMappingURL   string
	Timeout               time.Duration
	Verbose               bool
	VsockPort             uint32
}
