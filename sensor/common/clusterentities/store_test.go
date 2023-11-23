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
	ed.AddEndpoint(buildEndpoint(ip),
		EndpointTargetInfo{
			ContainerPort: port,
			PortName:      "ehlo",
		})
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
				{"10.0.0.1": false, "10.3.0.1": true}, // pre-tick 1: both must exist
				{"10.0.0.1": false, "10.3.0.1": true}, // tick 1: both must exist
				{"10.0.0.1": false, "10.3.0.1": true}, // tick 2: only the younger IP must exist
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
				{"10.0.0.1": false, "10.3.0.1": true}, // after-tick 1: only the younger IP must exist
				{"10.0.0.1": false, "10.3.0.1": true}, // after-tick 2: only the younger IP must exist
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
				{"10.0.0.1": true, "10.3.0.1": true},  // after-tick 1:  both must exist
				{"10.0.0.1": false, "10.3.0.1": true}, // after-tick 2: only the younger IP must exist
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
				for endpoint, shallExist := range expectation {
					result := entityStore.LookupByEndpoint(buildEndpoint(endpoint))
					if shallExist {
						s.True(len(result) > 0, "Should find endpoint %q in tick %d", endpoint, tickNo)
					} else {
						s.True(len(result) == 0, "Should not find endpoint %q in tick %d", endpoint, tickNo)
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
		// TODO(ROX-20716): Enable this test after fixing the issue of External Entities appearing for past IPs
		// "The store shall remember past IP of a container": {
		//	entityUpdates: []eUpdate{
		//		{
		//			containerID: "pod1",
		//			ipAddr:      "10.0.0.1",
		//			port:        80,
		//			incremental: true,
		//		},
		//		{
		//			containerID: "pod1",
		//			ipAddr:      "10.3.0.1",
		//			port:        80,
		//			incremental: false,
		//		},
		// 	},
		// 	expectedEndpoints: []string{"10.3.0.1", "10.0.0.1"},
		// },
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
