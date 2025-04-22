package manager

import (
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestNetworkFlowPurger(t *testing.T) {
	suite.Run(t, new(NetworkFlowPurgerTestSuite))
}

type NetworkFlowPurgerTestSuite struct {
	suite.Suite
}

func (s *NetworkFlowPurgerTestSuite) TestDisabledPurger() {
	purgerTickerC := make(chan time.Time)
	defer close(purgerTickerC)
	mockCtrl := gomock.NewController(s.T())
	enrichTickerC := make(chan time.Time)
	defer close(enrichTickerC)
	defer mockCtrl.Finish()
	s.T().Setenv(env.EnrichmentPurgerTickerCycle.EnvVar(), "0s")

	m, mockEntityStore, _, _ := createManager(mockCtrl, enrichTickerC)
	purger := NewNetworkFlowPurger(mockEntityStore, time.Hour, m, WithPurgerTicker(purgerTickerC))

	s.NoError(purger.Start())
	// ticking should not block
	purgerTickerC <- time.Now()
	// purgingDone should be signaled even if the purger does nothing
	s.Eventually(purger.purgingDone.IsDone, 500*time.Millisecond, 100*time.Millisecond)

	purger.Stop(nil)
}

func (s *NetworkFlowPurgerTestSuite) TestPurgerWithoutManager() {
	purgerTickerC := make(chan time.Time)
	defer close(purgerTickerC)
	mockCtrl := gomock.NewController(s.T())
	enrichTickerC := make(chan time.Time)
	defer close(enrichTickerC)
	defer mockCtrl.Finish()
	_, mockEntityStore, _, _ := createManager(mockCtrl, enrichTickerC)
	// Set manager to nil to explicitly simulate disconnected purger
	purger := NewNetworkFlowPurger(mockEntityStore, time.Hour, nil, WithPurgerTicker(purgerTickerC))

	s.Error(purger.Start())
	// Trigger the purger - shall not block despite the manager is missing
	purgerTickerC <- time.Now()
	// purgingDone should be signaled even if the purger does nothing
	s.Eventually(purger.purgingDone.IsDone, 500*time.Millisecond, 100*time.Millisecond)
	purger.Stop(nil)
}

func (s *NetworkFlowPurgerTestSuite) TestPurgerWithManager() {
	purgerTickerC := make(chan time.Time)
	defer close(purgerTickerC)

	const hostname = "host"
	mockCtrl := gomock.NewController(s.T())
	enrichTickerC := make(chan time.Time)
	defer close(enrichTickerC)
	defer mockCtrl.Finish()
	cases := map[string]struct {
		firstSeen            time.Duration
		lastUpdateTime       time.Duration
		purgerMaxAge         time.Duration
		isKnownEndpoint      bool
		expectedEndpoint     *containerEndpointIndicator
		expectedNumEndpoints int
	}{
		"Purger should purge something": {
			firstSeen:            2 * time.Hour,
			lastUpdateTime:       2 * time.Hour,
			purgerMaxAge:         1 * time.Hour,
			isKnownEndpoint:      true,
			expectedNumEndpoints: 0,
		},
		"Purger should purge nothing": {
			firstSeen:            2 * time.Hour,
			lastUpdateTime:       2 * time.Hour,
			purgerMaxAge:         3 * time.Hour,
			isKnownEndpoint:      true,
			expectedNumEndpoints: 1,
		},
	}
	for name, tc := range cases {
		s.Run(name, func() {
			now := time.Now()
			lastUpdateTS := timestamp.FromGoTime(now.Add(-tc.lastUpdateTime))

			m, mockEntityStore, _, _ := createManager(mockCtrl, enrichTickerC)
			purger := NewNetworkFlowPurger(mockEntityStore, tc.purgerMaxAge, m, WithPurgerTicker(purgerTickerC))

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
			s.Require().NoError(purger.Start())
			// Cycle through online-offline modes
			purger.Notify(common.SensorComponentEventOfflineMode)
			purger.Notify(common.SensorComponentEventCentralReachable)

			purgerTickerC <- time.Now()
			// wait until purger is done
			s.Require().Eventually(purger.purgingDone.IsDone, 2*time.Second, 500*time.Millisecond)
			s.Equal(tc.expectedNumEndpoints, len(m.connectionsByHost[hostname].endpoints))
			purger.Stop(nil)
		})
	}

}

func (s *NetworkFlowPurgerTestSuite) TestPurgerHostConnsEndpoints() {
	const hostname = "host"
	mockCtrl := gomock.NewController(s.T())
	enrichTickerC := make(chan time.Time)
	defer close(enrichTickerC)
	defer mockCtrl.Finish()
	id := "id"
	_ = id
	cases := map[string]struct {
		firstSeen         time.Duration
		lastUpdateTime    time.Duration
		purgerMaxAge      time.Duration
		isKnownEndpoint   bool
		expectedEndpoint  *containerEndpointIndicator
		expectedPurgedEps int
	}{
		"Endpoints-maxAge: should purge old endpoints": {
			firstSeen:         2 * time.Hour,
			lastUpdateTime:    2 * time.Hour,
			purgerMaxAge:      time.Hour,
			isKnownEndpoint:   true,
			expectedPurgedEps: 1,
		},
		"Endpoints-maxAge: should keep endpoints with young lastUpdateTime": {
			firstSeen:         time.Minute,
			lastUpdateTime:    time.Minute,
			purgerMaxAge:      time.Hour,
			isKnownEndpoint:   true,
			expectedPurgedEps: 0,
		},
		"Endpoints-endpoint-gone: should remove unknown endpoints": {
			firstSeen:         time.Minute,
			lastUpdateTime:    time.Minute,
			purgerMaxAge:      time.Hour,
			isKnownEndpoint:   false,
			expectedPurgedEps: 1,
		},
	}
	for name, tc := range cases {
		s.Run(name, func() {
			now := time.Now()
			lastUpdateTS := timestamp.FromGoTime(now.Add(-tc.lastUpdateTime))

			m, mockEntityStore, _, _ := createManager(mockCtrl, enrichTickerC)
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
			npe, _ := purgeHostConns(&m.connectionsByHostMutex, tc.purgerMaxAge, m.connectionsByHost, m.clusterEntities)
			s.Equal(tc.expectedPurgedEps, npe)
		})
	}
}

func (s *NetworkFlowPurgerTestSuite) TestPurgerHostConnsConnections() {
	const hostname = "host"
	mockCtrl := gomock.NewController(s.T())
	enrichTickerC := make(chan time.Time)
	defer close(enrichTickerC)
	defer mockCtrl.Finish()
	id := "id"
	_ = id
	cases := map[string]struct {
		firstSeen             time.Duration
		lastUpdateTime        time.Duration
		purgerMaxAge          time.Duration
		foundContainerID      bool
		containerIDHistorical bool
		expectedPurgedConns   int
	}{
		"Connections-maxAge: should purge old connections": {
			firstSeen:           2 * time.Hour,
			lastUpdateTime:      2 * time.Hour,
			purgerMaxAge:        time.Hour,
			foundContainerID:    true,
			expectedPurgedConns: 1,
		},
		"Connections-maxAge: should keep connections with young lastUpdateTime": {
			firstSeen:           time.Minute,
			lastUpdateTime:      time.Minute,
			purgerMaxAge:        time.Hour,
			foundContainerID:    true,
			expectedPurgedConns: 0,
		},
		"Connections-containerID-gone: should keep connections related to historical containers": {
			firstSeen:             time.Minute,
			lastUpdateTime:        time.Minute,
			purgerMaxAge:          time.Hour,
			foundContainerID:      true,
			containerIDHistorical: true,
			expectedPurgedConns:   0,
		},
		"Connections-containerID-gone: should remove connections with unknown endpoints": {
			firstSeen:           time.Minute,
			lastUpdateTime:      time.Minute,
			purgerMaxAge:        time.Hour,
			foundContainerID:    false,
			expectedPurgedConns: 1,
		},
	}
	for name, tc := range cases {
		s.Run(name, func() {
			now := time.Now()
			lastUpdateTS := timestamp.FromGoTime(now.Add(-tc.lastUpdateTime))

			m, mockEntityStore, _, _ := createManager(mockCtrl, enrichTickerC)
			expectationsEndpointPurger(mockEntityStore, true, tc.foundContainerID, tc.containerIDHistorical)

			pair := createConnectionPair().
				firstSeen(timestamp.FromGoTime(now.Add(-tc.firstSeen))).
				tsAdded(lastUpdateTS)
			m.activeConnections[*pair.conn] = &networkConnIndicatorWithAge{lastUpdate: lastUpdateTS}
			addHostConnection(m, createHostnameConnections(hostname).withConnectionPair(pair))

			_, npc := purgeHostConns(&m.connectionsByHostMutex, tc.purgerMaxAge, m.connectionsByHost, m.clusterEntities)
			s.Equal(tc.expectedPurgedConns, npc)
		})
	}
}

func (s *NetworkFlowPurgerTestSuite) TestPurgerActiveConnections() {
	mockCtrl := gomock.NewController(s.T())
	enrichTickerC := make(chan time.Time)
	defer close(enrichTickerC)
	defer mockCtrl.Finish()
	id := "id"
	_ = id
	cases := map[string]struct {
		firstSeen             time.Duration
		lastUpdateTime        time.Duration
		purgerMaxAge          time.Duration
		foundContainerID      bool
		containerIDHistorical bool
		expectedPurgedConns   int
	}{
		"Connections-maxAge: should purge old connections": {
			firstSeen:           2 * time.Hour,
			lastUpdateTime:      2 * time.Hour,
			purgerMaxAge:        time.Hour,
			foundContainerID:    true,
			expectedPurgedConns: 1,
		},
		"Connections-maxAge: should keep connections with young lastUpdateTime": {
			firstSeen:           time.Minute,
			lastUpdateTime:      time.Minute,
			purgerMaxAge:        time.Hour,
			foundContainerID:    true,
			expectedPurgedConns: 0,
		},
		"Connections-containerID-gone: should keep connections related to historical containers": {
			firstSeen:             time.Minute,
			lastUpdateTime:        time.Minute,
			purgerMaxAge:          time.Hour,
			foundContainerID:      true,
			containerIDHistorical: true,
			expectedPurgedConns:   0,
		},
		"Connections-containerID-gone: should remove connections with unknown endpoints": {
			firstSeen:           time.Minute,
			lastUpdateTime:      time.Minute,
			purgerMaxAge:        time.Hour,
			foundContainerID:    false,
			expectedPurgedConns: 1,
		},
	}
	for name, tc := range cases {
		s.Run(name, func() {
			now := time.Now()
			lastUpdateTS := timestamp.FromGoTime(now.Add(-tc.lastUpdateTime))

			m, mockEntityStore, _, _ := createManager(mockCtrl, enrichTickerC)
			expectationsEndpointPurger(mockEntityStore, true, tc.foundContainerID, tc.containerIDHistorical)

			pair := createConnectionPair().
				firstSeen(timestamp.FromGoTime(now.Add(-tc.firstSeen))).
				tsAdded(lastUpdateTS)
			dummy := sync.Mutex{}
			activeConns := map[connection]*networkConnIndicatorWithAge{
				*pair.conn: {lastUpdate: lastUpdateTS},
			}

			npc := purgeActiveConnections(&dummy, tc.purgerMaxAge, activeConns, m.clusterEntities)
			s.Equal(tc.expectedPurgedConns, npc)
		})
	}
}

func (s *NetworkFlowPurgerTestSuite) TestPurgerActiveEndpoints() {
	mockCtrl := gomock.NewController(s.T())
	enrichTickerC := make(chan time.Time)
	defer close(enrichTickerC)
	defer mockCtrl.Finish()
	id := "id"
	_ = id
	cases := map[string]struct {
		firstSeen             time.Duration
		lastUpdateTime        time.Duration
		purgerMaxAge          time.Duration
		isKnownEndpoint       bool
		foundContainerID      bool
		containerIDHistorical bool
		expectedPurgedEps     int
	}{
		"Endpoints-maxAge: should purge old endpoints": {
			firstSeen:         2 * time.Hour,
			lastUpdateTime:    2 * time.Hour,
			purgerMaxAge:      time.Hour,
			foundContainerID:  true,
			isKnownEndpoint:   true,
			expectedPurgedEps: 1,
		},
		"Endpoints-maxAge: should keep endpoints with young lastUpdateTime": {
			firstSeen:         time.Minute,
			lastUpdateTime:    time.Minute,
			purgerMaxAge:      time.Hour,
			foundContainerID:  true,
			isKnownEndpoint:   true,
			expectedPurgedEps: 0,
		},
		"Endpoints-containerID-gone: should keep endpoints related to historical containers": {
			firstSeen:             time.Minute,
			lastUpdateTime:        time.Minute,
			purgerMaxAge:          time.Hour,
			foundContainerID:      true,
			containerIDHistorical: true,
			isKnownEndpoint:       true,
			expectedPurgedEps:     0,
		},
		"Endpoints-containerID-gone: should purge endpoints related to unknown containers": {
			firstSeen:             time.Minute,
			lastUpdateTime:        time.Minute,
			purgerMaxAge:          time.Hour,
			foundContainerID:      false,
			containerIDHistorical: false,
			isKnownEndpoint:       true,
			expectedPurgedEps:     1,
		},
		"Endpoints-endpoint-gone: should remove endpoints with unknown endpoints": {
			firstSeen:         time.Minute,
			lastUpdateTime:    time.Minute,
			purgerMaxAge:      time.Hour,
			isKnownEndpoint:   false,
			expectedPurgedEps: 1,
		},
	}
	for name, tc := range cases {
		s.Run(name, func() {
			now := time.Now()
			lastUpdateTS := timestamp.FromGoTime(now.Add(-tc.lastUpdateTime))

			m, mockEntityStore, _, _ := createManager(mockCtrl, enrichTickerC)
			expectationsEndpointPurger(mockEntityStore, tc.isKnownEndpoint, tc.foundContainerID, tc.containerIDHistorical)

			ep := createEndpointPair(timestamp.FromGoTime(now.Add(-tc.firstSeen)), lastUpdateTS)
			dummy := sync.Mutex{}
			activeEndpoints := map[containerEndpoint]*containerEndpointIndicatorWithAge{
				*ep.endpoint: {lastUpdate: lastUpdateTS},
			}

			npe := purgeActiveEndpoints(&dummy, tc.purgerMaxAge, activeEndpoints, m.clusterEntities)
			s.Equal(tc.expectedPurgedEps, npe)
		})
	}
}
