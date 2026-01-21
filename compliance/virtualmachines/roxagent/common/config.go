package common

import "time"

type Config struct {
	DaemonMode          bool
	IndexHostPath       string
	IndexInterval       time.Duration
	SendNow             bool
	RepoToCPEMappingURL string
	Timeout             time.Duration
	Verbose             bool
	VsockPort           uint32
}
