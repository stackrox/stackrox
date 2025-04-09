package manager

import (
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timestamp"
	"go.uber.org/mock/gomock"
)

func (s *NetworkFlowManagerTestSuite) TestPurgerHostConnsEndpoints() {
	const hostname = "host"
	mockCtrl := gomock.NewController(s.T())
	enrichTickerC := make(chan time.Time)
	purgerTickerC := make(chan time.Time)
	defer close(enrichTickerC)
	defer close(purgerTickerC)
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

			m, mockEntityStore, _, _ := createManager(mockCtrl, enrichTickerC, purgerTickerC)
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

func (s *NetworkFlowManagerTestSuite) TestPurgerHostConnsConnections() {
	const hostname = "host"
	mockCtrl := gomock.NewController(s.T())
	enrichTickerC := make(chan time.Time)
	purgerTickerC := make(chan time.Time)
	defer close(enrichTickerC)
	defer close(purgerTickerC)
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

			m, mockEntityStore, _, _ := createManager(mockCtrl, enrichTickerC, purgerTickerC)
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

func (s *NetworkFlowManagerTestSuite) TestPurgerActiveConnections() {
	mockCtrl := gomock.NewController(s.T())
	enrichTickerC := make(chan time.Time)
	purgerTickerC := make(chan time.Time)
	defer close(enrichTickerC)
	defer close(purgerTickerC)
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

			m, mockEntityStore, _, _ := createManager(mockCtrl, enrichTickerC, purgerTickerC)
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

func (s *NetworkFlowManagerTestSuite) TestPurgerActiveEndpoints() {
	mockCtrl := gomock.NewController(s.T())
	enrichTickerC := make(chan time.Time)
	purgerTickerC := make(chan time.Time)
	defer close(enrichTickerC)
	defer close(purgerTickerC)
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

			m, mockEntityStore, _, _ := createManager(mockCtrl, enrichTickerC, purgerTickerC)
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
