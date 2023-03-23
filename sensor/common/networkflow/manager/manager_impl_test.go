package manager

import (
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stretchr/testify/suite"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
)

var (
	openNetworkEndpoint = &sensor.NetworkEndpoint{
					SocketFamily: sensor.SocketFamily_SOCKET_FAMILY_IPV4,
					Protocol:	storage.L4Protocol_L4_PROTOCOL_TCP,
					ContainerId:	"FakeContainerId",
					ListenAddress:	&sensor.NetworkAddress{
						Port:	80,
					},
					Originator:	&storage.NetworkProcessUniqueKey{
						ProcessName:	"socat",
						ProcessExecFilePath:	"/usr/bin/socat",
						ProcessArgs:		"port: 80",
					},
				}
	closedNetworkEndpoint = &sensor.NetworkEndpoint{
					SocketFamily: sensor.SocketFamily_SOCKET_FAMILY_IPV4,
					Protocol:	storage.L4Protocol_L4_PROTOCOL_TCP,
					ContainerId:	"FakeContainerId",
					CloseTimestamp: protoconv.ConvertTimeToTimestamp(time.Now()),
					ListenAddress:	&sensor.NetworkAddress{
						Port:	80,
					},
					Originator:	&storage.NetworkProcessUniqueKey{
						ProcessName:	"socat",
						ProcessExecFilePath:	"/usr/bin/socat",
						ProcessArgs:		"port: 80",
					},
				}
)

func TestNetworkflowManager(t *testing.T) {
	suite.Run(t, new(NetworkflowManagerTestSuite))
}

type NetworkflowManagerTestSuite struct {
	suite.Suite
}

func (suite *NetworkflowManagerTestSuite) TestAddNothing() {
	h := hostConnections{}
	h.endpoints = make(map[containerEndpoint]*connStatus)

	networkInfo := &sensor.NetworkConnectionInfo{}
	nowTimestamp := timestamp.Now()
	var sequenceID int64 = 0
	err := h.Process(networkInfo, nowTimestamp, sequenceID)
	suite.NoError(err)
	suite.Len(h.endpoints, 0)
}

func (suite *NetworkflowManagerTestSuite) TestAddOpen() {
	h := hostConnections{}
	h.endpoints = make(map[containerEndpoint]*connStatus)

	networkInfo := &sensor.NetworkConnectionInfo{
		UpdatedEndpoints: []*sensor.NetworkEndpoint{openNetworkEndpoint},
	}

	nowTimestamp := timestamp.Now()
	var sequenceID int64 = 0
	h.connectionsSequenceID = sequenceID
	err := h.Process(networkInfo, nowTimestamp, sequenceID)
	suite.NoError(err)
	suite.Len(h.endpoints, 1)
}

func (suite *NetworkflowManagerTestSuite) TestAddOpenAndClosed() {
	h := hostConnections{}
	h.endpoints = make(map[containerEndpoint]*connStatus)

	networkInfoOpen := &sensor.NetworkConnectionInfo{
		UpdatedEndpoints: []*sensor.NetworkEndpoint{openNetworkEndpoint},
	}

	networkInfoClosed := &sensor.NetworkConnectionInfo{
		UpdatedEndpoints: []*sensor.NetworkEndpoint{closedNetworkEndpoint},
	}

	nowTimestamp := timestamp.Now()
	var sequenceID int64 = 0
	h.connectionsSequenceID = sequenceID

	err := h.Process(networkInfoOpen, nowTimestamp, sequenceID)
	suite.NoError(err)

	err = h.Process(networkInfoClosed, nowTimestamp, sequenceID)
	suite.NoError(err)

	suite.Len(h.endpoints, 1)
}
