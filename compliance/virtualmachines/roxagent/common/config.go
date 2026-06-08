package common

import (
	"time"

	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
)

type Config struct {
	DaemonMode            bool
	IndexHostPath         string
	IndexInterval         time.Duration
	MaxInitialReportDelay time.Duration
	RepoToCPEMappingURL   string
	Timeout               time.Duration
	Trigger               v1.ReportTrigger
	Verbose               bool
	VsockPort             uint32
}
