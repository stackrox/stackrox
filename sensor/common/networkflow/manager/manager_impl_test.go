package manager

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stretchr/testify/suite"
)

var (
	openNetworkEndpoint = &sensor.NetworkEndpoint{
		SocketFamily: sensor.SocketFamily_SOCKET_FAMILY_IPV4,
		Protocol:     storage.L4Protocol_L4_PROTOCOL_TCP,
		ContainerId:  "FakeContainerId",
		ListenAddress: &sensor.NetworkAddress{
			Port: 80,
		},
		Originator: &storage.NetworkProcessUniqueKey{
			ProcessName:         "socat",
			ProcessExecFilePath: "/usr/bin/socat",
			ProcessArgs:         "port: 80",
		},
	}
	openNetworkEndpoint81 = &sensor.NetworkEndpoint{
		SocketFamily: sensor.SocketFamily_SOCKET_FAMILY_IPV4,
		Protocol:     storage.L4Protocol_L4_PROTOCOL_TCP,
		ContainerId:  "FakeContainerId",
		ListenAddress: &sensor.NetworkAddress{
			Port: 81,
		},
		Originator: &storage.NetworkProcessUniqueKey{
			ProcessName:         "socat",
			ProcessExecFilePath: "/usr/bin/socat",
			ProcessArgs:         "port: 81",
		},
	}
	openNetworkEndpointNoOriginator = &sensor.NetworkEndpoint{
		SocketFamily: sensor.SocketFamily_SOCKET_FAMILY_IPV4,
		Protocol:     storage.L4Protocol_L4_PROTOCOL_TCP,
		ContainerId:  "FakeContainerId",
		ListenAddress: &sensor.NetworkAddress{
			Port: 80,
		},
	}
	closedNetworkEndpoint = &sensor.NetworkEndpoint{
		SocketFamily:   sensor.SocketFamily_SOCKET_FAMILY_IPV4,
		Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
		ContainerId:    "FakeContainerId",
		CloseTimestamp: protoconv.ConvertTimeToTimestamp(time.Now()),
		ListenAddress: &sensor.NetworkAddress{
			Port: 80,
		},
		Originator: &storage.NetworkProcessUniqueKey{
			ProcessName:         "socat",
			ProcessExecFilePath: "/usr/bin/socat",
			ProcessArgs:         "port: 80",
		},
	}
)

func TestNetworkFlowManager(t *testing.T) {
	suite.Run(t, new(NetworkFlowManagerTestSuite))
}

type NetworkFlowManagerTestSuite struct {
	suite.Suite
}

func (s *NetworkFlowManagerTestSuite) TestAddNothing() {
	h := hostConnections{}
	h.endpoints = make(map[containerEndpoint]*connStatus)

	networkInfo := &sensor.NetworkConnectionInfo{}
	nowTimestamp := timestamp.Now()
	var sequenceID int64
	err := h.Process(networkInfo, nowTimestamp, sequenceID)
	s.NoError(err)
	s.Len(h.endpoints, 0)
}

func (s *NetworkFlowManagerTestSuite) TestAddOpen() {
	h := hostConnections{}
	h.endpoints = make(map[containerEndpoint]*connStatus)

	networkInfo := &sensor.NetworkConnectionInfo{
		UpdatedEndpoints: []*sensor.NetworkEndpoint{openNetworkEndpoint},
	}

	nowTimestamp := timestamp.Now()
	var sequenceID int64
	h.connectionsSequenceID = sequenceID
	err := h.Process(networkInfo, nowTimestamp, sequenceID)
	s.NoError(err)
	s.Len(h.endpoints, 1)
}

func (s *NetworkFlowManagerTestSuite) TestAddOpenAndClosed() {
	h := hostConnections{}
	h.endpoints = make(map[containerEndpoint]*connStatus)

	networkInfoOpen := &sensor.NetworkConnectionInfo{
		UpdatedEndpoints: []*sensor.NetworkEndpoint{openNetworkEndpoint},
	}

	networkInfoClosed := &sensor.NetworkConnectionInfo{
		UpdatedEndpoints: []*sensor.NetworkEndpoint{closedNetworkEndpoint},
	}

	nowTimestamp := timestamp.Now()
	var sequenceID int64
	h.connectionsSequenceID = sequenceID

	err := h.Process(networkInfoOpen, nowTimestamp, sequenceID)
	s.NoError(err)

	err = h.Process(networkInfoClosed, nowTimestamp, sequenceID)
	s.NoError(err)

	s.Len(h.endpoints, 1)
}

func (s *NetworkFlowManagerTestSuite) TestAddTwoDifferent() {
	h := hostConnections{}
	h.endpoints = make(map[containerEndpoint]*connStatus)

	networkInfoOpen := &sensor.NetworkConnectionInfo{
		UpdatedEndpoints: []*sensor.NetworkEndpoint{openNetworkEndpoint},
	}

	networkInfoOpen81 := &sensor.NetworkConnectionInfo{
		UpdatedEndpoints: []*sensor.NetworkEndpoint{openNetworkEndpoint81},
	}

	nowTimestamp := timestamp.Now()
	var sequenceID int64
	h.connectionsSequenceID = sequenceID

	err := h.Process(networkInfoOpen, nowTimestamp, sequenceID)
	s.NoError(err)

	err = h.Process(networkInfoOpen81, nowTimestamp, sequenceID)
	s.NoError(err)

	s.Len(h.endpoints, 2)
}

func (s *NetworkFlowManagerTestSuite) TestAddTwoDifferentSameBatch() {
	h := hostConnections{}
	h.endpoints = make(map[containerEndpoint]*connStatus)

	networkInfoOpen := &sensor.NetworkConnectionInfo{
		UpdatedEndpoints: []*sensor.NetworkEndpoint{openNetworkEndpoint, openNetworkEndpoint81},
	}

	nowTimestamp := timestamp.Now()
	var sequenceID int64
	h.connectionsSequenceID = sequenceID

	err := h.Process(networkInfoOpen, nowTimestamp, sequenceID)
	s.NoError(err)

	s.Len(h.endpoints, 2)
}

func (s *NetworkFlowManagerTestSuite) TestAddNoOriginator() {
	h := hostConnections{}
	h.endpoints = make(map[containerEndpoint]*connStatus)

	networkInfoOpen := &sensor.NetworkConnectionInfo{
		UpdatedEndpoints: []*sensor.NetworkEndpoint{openNetworkEndpointNoOriginator},
	}

	nowTimestamp := timestamp.Now()
	var sequenceID int64
	h.connectionsSequenceID = sequenceID

	err := h.Process(networkInfoOpen, nowTimestamp, sequenceID)
	s.NoError(err)

	s.Len(h.endpoints, 1)
}
