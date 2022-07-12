package metrics

import (
	"time"

	"github.com/stackrox/rox/pkg/metrics"
)

// SetPostgresOperationDurationTime mock
func SetPostgresOperationDurationTime(_ time.Time, _ metrics.Op, _ string) {}

// SetAcquireDBConnDuration mock
func SetAcquireDBConnDuration(_ time.Time, _ metrics.Op, _ string) {}

// SetBoltOperationDurationTime mock
func SetBoltOperationDurationTime(_ time.Time, _ metrics.Op, _ string) {}

// SetDackboxOperationDurationTime mock
func SetDackboxOperationDurationTime(_ time.Time, _ metrics.Op, _ string) {}
