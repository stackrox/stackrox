package updatecomputer

import (
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common/networkflow/manager/indicator"
)

// EndpointDeduperAssertion is a function that can inspect deduper state for testing
type EndpointDeduperAssertion func(map[indicator.BinaryHash]indicator.BinaryHash)

// TestableUpdateComputer extends UpdateComputer with testing capabilities.
// It is used in tests to assert on the deduper state.
type TestableUpdateComputer interface {
	UpdateComputer
	// WithEndpointDeduperAccess executes the assertion with access to internal deduper state
	WithEndpointDeduperAccess(assertion EndpointDeduperAssertion)
}

// WithEndpointDeduperAccess executes the assertion with access to internal endpoint deduper state
func (c *TransitionBased) WithEndpointDeduperAccess(assertion EndpointDeduperAssertion) {
	// Provide direct access to the binary hash deduper
	concurrency.WithRLock(&c.endpointsDeduperMutex, func() {
		assertion(c.endpointsDeduper)
	})
}
