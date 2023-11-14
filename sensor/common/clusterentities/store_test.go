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
		//	},
		//	expectedEndpoints: []string{"10.3.0.1", "10.0.0.1"},
		//},
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
