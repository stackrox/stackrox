package manager

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/timestamp"
	"go.uber.org/mock/gomock"
)

func (s *NetworkFlowManagerTestSuite) TestEndpointPurger() {
	const hostname = "host"
	mockCtrl := gomock.NewController(s.T())
	enrichTickerC := make(chan time.Time)
	purgerTickerC := make(chan time.Time)
	defer close(enrichTickerC)
	defer close(purgerTickerC)
	defer mockCtrl.Finish()
	m, mockEntityStore, _, _ := createManager(mockCtrl, enrichTickerC, purgerTickerC)
	id := "id"
	_ = id
	cases := map[string]struct {
		firstSeen            time.Duration
		lastUpdateTime       time.Duration
		purgerMaxAge         time.Duration
		isKnownEndpoint      bool
		expectedStatus       *connStatus
		expectedEndpoint     *containerEndpointIndicator
		expectedHostConnSize int
	}{
		"Purger maxAge: should purge old endpoints": {
			firstSeen:            2 * time.Hour,
			lastUpdateTime:       2 * time.Hour,
			purgerMaxAge:         time.Hour,
			isKnownEndpoint:      true,
			expectedHostConnSize: 0,
		},
		"Purger maxAge: should keep endpoints with young lastUpdateTime": {
			firstSeen:            time.Minute,
			lastUpdateTime:       time.Minute,
			purgerMaxAge:         time.Hour,
			isKnownEndpoint:      true,
			expectedHostConnSize: 1,
		},
		"Purger endpoint-gone: should remove unknown endpoints": {
			firstSeen:            time.Minute,
			lastUpdateTime:       time.Minute,
			purgerMaxAge:         time.Hour,
			isKnownEndpoint:      false,
			expectedHostConnSize: 1,
		},
	}
	for name, tc := range cases {
		s.Run(name, func() {
			now := time.Now()
			lastUpdateTS := timestamp.FromGoTime(now.Add(-tc.lastUpdateTime))
			expectationsEndpointPurger(mockEntityStore, tc.isKnownEndpoint, true, false)
			ep := createEndpointPair(timestamp.FromGoTime(now.Add(-tc.firstSeen)), lastUpdateTS)
			concurrency.WithLock(&m.connectionsByHostMutex, func() {
				m.connectionsByHost[hostname] = &hostConnections{
					hostname:    hostname,
					connections: nil,
					endpoints: map[containerEndpoint]*connStatus{
						*ep.endpoint: ep.status,
					},
				}
			})
			// Purger checks activeEndpoints only if not empty, so let's make sure that
			// the mock is called correct number of times by always having one active endpoint.
			m.activeEndpoints[*ep.endpoint] = &containerEndpointIndicatorWithAge{
				containerEndpointIndicator: containerEndpointIndicator{
					entity:   networkgraph.Entity{},
					port:     80,
					protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
				},
				lastUpdate: lastUpdateTS,
			}
			m.runAllPurgerRules(tc.purgerMaxAge)

			concurrency.WithLock(&m.connectionsByHostMutex, func() {
				s.Len(m.connectionsByHost[hostname].endpoints, tc.expectedHostConnSize)
			})
		})
	}
}
