package updatecomputer

import (
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common/networkflow/manager/indicator"
)

// EndpointDeduperAssertion is a function that can inspect deduper state for testing
type EndpointDeduperAssertion func(map[indicator.BinaryHash]indicator.BinaryHash)

// WithEndpointDeduperAccess executes the assertion with access to internal endpoint deduper state
func (c *Computer) WithEndpointDeduperAccess(assertion EndpointDeduperAssertion) {
	// Provide direct access to the binary hash deduper
	concurrency.WithRLock(&c.endpointsDeduperMutex, func() {
		assertion(c.endpointsDeduper)
	})
}
