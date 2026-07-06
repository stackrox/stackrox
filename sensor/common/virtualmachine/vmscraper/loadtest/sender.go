package loadtest

import (
	"context"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/sensor/common/virtualmachine"
)

// NullSender implements vmscraper.IndexReportSender as a no-op stand-in for
// Central, which is explicitly out of scope for this investigation.
//
// intentional simplification: no rate limiting or latency modeling. Upgrade
// path: add an optional token-bucket limiter (mirroring
// ROX_VM_INDEX_REPORT_RATE_LIMIT) if Central-side backpressure ever needs to
// be observed on Sensor -- see design spec Part A, NullSender section.
type NullSender struct{}

// Send implements vmscraper.IndexReportSender.
func (NullSender) Send(_ context.Context, _ *virtualmachine.Info, _ *v4.IndexReport) error {
	return nil
}
