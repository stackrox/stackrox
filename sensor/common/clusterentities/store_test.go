package clusterentities

import (
	"testing"

	"github.com/stackrox/rox/pkg/net"
	"github.com/stretchr/testify/suite"
)

func TestClusterEntitiesStore(t *testing.T) {
	suite.Run(t, new(ClusterEntitiesStoreTestSuite))
}

type ClusterEntitiesStoreTestSuite struct {
	suite.Suite
}

// region external-entities test

func buildEndpoint(ip string) net.NumericEndpoint {
	return net.NumericEndpoint{
		IPAndPort: net.NetworkPeerID{
			Address: net.ParseIP(ip),
		},
		L4Proto: net.TCP,
	}
}

func entityUpdate(ip string, port uint16) *EntityData {
	ed := &EntityData{}
	ep := buildEndpoint(ip)
	ed.AddEndpoint(ep, EndpointTargetInfo{
		ContainerPort: port,
		PortName:      "ehlo",
	})
	ed.AddIP(ep.IPAndPort.Address)
	return ed
}

func (s *ClusterEntitiesStoreTestSuite) TestMemoryAboutPast() {
	type eUpdate struct {
		containerID string
		ipAddr      string
		port        uint16
		incremental bool
	}
	cases := map[string]struct {
		numTicksToRemember uint16
		entityUpdates      []eUpdate
		endpointsAfterTick []map[string]bool
	}{
		"Memory disabled should forget 10.0.0.1 immediately": {
			numTicksToRemember: 0,
			entityUpdates: []eUpdate{
				{
					containerID: "pod1",
					ipAddr:      "10.0.0.1",
					port:        80,
					incremental: true, // append
				},
				{
					containerID: "pod1",
					ipAddr:      "10.3.0.1",
					port:        80,
					incremental: false, // replace
				},
			},
			endpointsAfterTick: []map[string]bool{
				{"10.0.0.1": false, "10.3.0.1": true}, // pre-tick 1: 10.0.0.1 should be overwritten immediately - only 10.3.0.1 should exist
				{"10.0.0.1": false, "10.3.0.1": true}, // tick 1: only 10.3.0.1 should exist
				{"10.0.0.1": false, "10.3.0.1": true}, // tick 2: only 10.3.0.1 should exist
			},
		},
		"Old IPs should be gone on the first tick": {
			numTicksToRemember: 1,
			entityUpdates: []eUpdate{
				{
					containerID: "pod1",
					ipAddr:      "10.0.0.1",
					port:        80,
					incremental: true,
				},
				{
					containerID: "pod1",
					ipAddr:      "10.3.0.1",
					port:        80,
					incremental: false,
				},
			},
			endpointsAfterTick: []map[string]bool{
				{"10.0.0.1": true, "10.3.0.1": true},  // pre-tick 1: both must exist
				{"10.0.0.1": false, "10.3.0.1": true}, // after-tick 1: only 10.3.0.1 should exist
				{"10.0.0.1": false, "10.3.0.1": true}, // after-tick 2: only 10.3.0.1 should exist
			},
		},
		"Old IPs should be gone on the 2nd tick": {
			numTicksToRemember: 2,
			entityUpdates: []eUpdate{
				{
					containerID: "pod1",
					ipAddr:      "10.0.0.1",
					port:        80,
					incremental: true,
				},
				{
					containerID: "pod1",
					ipAddr:      "10.3.0.1",
					port:        80,
					incremental: false,
				},
			},
			endpointsAfterTick: []map[string]bool{
				{"10.0.0.1": true, "10.3.0.1": true},  // pre-tick 1: both must exist
				{"10.0.0.1": true, "10.3.0.1": true},  // after-tick 1: both must exist
				{"10.0.0.1": false, "10.3.0.1": true}, // after-tick 2: only 10.3.0.1 should exist
			},
		},
		"Old IPs should be gone for selected pods only": {
			numTicksToRemember: 2,
			entityUpdates: []eUpdate{
				{
					containerID: "pod1",
					ipAddr:      "10.0.0.1",
					port:        80,
					incremental: true,
				},
				{
					containerID: "pod1",
					ipAddr:      "10.3.0.1",
					port:        80,
					incremental: false,
				},
				{
					containerID: "pod2",
					ipAddr:      "20.0.0.1",
					port:        80,
					incremental: true,
				},
				{
					containerID: "pod2",
					ipAddr:      "20.3.0.1",
					port:        80,
					incremental: true,
				},
			},
			endpointsAfterTick: []map[string]bool{
				{"10.0.0.1": true, "10.3.0.1": true, "20.0.0.1": true, "20.3.0.1": true},
				{"10.0.0.1": true, "10.3.0.1": true, "20.0.0.1": true, "20.3.0.1": true},
				{"10.0.0.1": false, "10.3.0.1": true, "20.0.0.1": true, "20.3.0.1": true},
			},
		},
	}
	for name, tCase := range cases {
		s.Run(name, func() {
			entityStore := NewStoreWithMemory(tCase.numTicksToRemember)
			// Entities are updated based on the data from K8s
			for _, update := range tCase.entityUpdates {
				entityStore.Apply(map[string]*EntityData{
					update.containerID: entityUpdate(update.ipAddr, update.port),
				}, update.incremental)
			}

			for tickNo, expectation := range tCase.endpointsAfterTick {
				s.T().Logf("Historical IPs (tick %d): %v", tickNo, entityStore.historicalIPs)
				s.T().Logf("All IPs (tick %d): %v", tickNo, entityStore.ipMap)
				for endpoint, shallExist := range expectation {
					result := entityStore.LookupByEndpoint(buildEndpoint(endpoint))
					if shallExist {
						s.True(len(result) > 0, "Should find endpoint %q in tick %d. Result: %v", endpoint, tickNo, result)
					} else {
						s.True(len(result) == 0, "Should not find endpoint %q in tick %d.  Result: %v", endpoint, tickNo, result)
					}
				}
				entityStore.Tick()
			}
		})
	}
}

func (s *ClusterEntitiesStoreTestSuite) TestChangingIPsAndExternalEntities() {
	entityStore := NewStore()
	type eUpdate struct {
		containerID string
		ipAddr      string
		port        uint16
		incremental bool
	}
	cases := map[string]struct {
		entityUpdates     []eUpdate
		expectedEndpoints []string
	}{
		"Incremental updates to the store shall not loose data": {
			entityUpdates: []eUpdate{
				{
					containerID: "pod1",
					ipAddr:      "10.0.0.1",
					port:        80,
					incremental: true,
				},
				{
					containerID: "pod1",
					ipAddr:      "10.3.0.1",
					port:        80,
					incremental: true,
				},
			},
			expectedEndpoints: []string{"10.0.0.1", "10.3.0.1"},
		},
		"Non-incremental updates to the store shall overwrite all data for a key": {
			entityUpdates: []eUpdate{
				{
					containerID: "pod1",
					ipAddr:      "10.0.0.1",
					port:        80,
					incremental: true,
				},
				{
					containerID: "pod1",
					ipAddr:      "10.3.0.1",
					port:        80,
					incremental: false,
				},
				{
					containerID: "pod2",
					ipAddr:      "10.0.0.2",
					port:        80,
					incremental: true,
				},
			},
			expectedEndpoints: []string{"10.3.0.1", "10.0.0.2"},
		},
	}
	for name, tCase := range cases {
		s.Run(name, func() {
			for _, update := range tCase.entityUpdates {
				entityStore.Apply(map[string]*EntityData{
					update.containerID: entityUpdate(update.ipAddr, update.port),
				}, update.incremental)
			}
			for _, expectedEndpoint := range tCase.expectedEndpoints {
				result := entityStore.LookupByEndpoint(buildEndpoint(expectedEndpoint))
				s.Lenf(result, 1, "Expected endpoint %q not found", expectedEndpoint)
			}
		})
	}
}

// endregion
