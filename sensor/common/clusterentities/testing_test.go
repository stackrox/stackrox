package clusterentities

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/set"

	"golang.org/x/exp/maps"
)

type operation string

const (
	mapReset operation = "mapReset"
	// for simplicity of the test, we assume that all delete request will be for depl1
	deleteDeployment1 operation = "deleteDeployment1"
)

func buildEndpoint(ip string) net.NumericEndpoint {
	return net.NumericEndpoint{
		IPAndPort: net.NetworkPeerID{
			Address: net.ParseIP(ip),
		},
		L4Proto: net.TCP,
	}
}
func entityUpdate(ip, contID string, port uint16) *EntityData {
	// Check if this is a DELETE request
	if ip == "" && contID == "" && port == 0 {
		return &EntityData{}
	}
	return entityUpdateWithPortName(ip, contID, port, "http")
}

func entityUpdateWithPortName(ip, contID string, port uint16, portName string) *EntityData {
	ed := &EntityData{}
	ep := buildEndpoint(ip)
	ed.AddEndpoint(ep, EndpointTargetInfo{
		ContainerPort: port,
		PortName:      portName,
	})
	ed.AddIP(ep.IPAndPort.Address)
	ed.AddContainerID(contID, ContainerMetadata{
		DeploymentID:  "",
		DeploymentTS:  0,
		PodID:         "",
		PodUID:        "",
		ContainerName: "name-of-" + contID,
		ContainerID:   contID,
		Namespace:     "",
		StartTime:     nil,
		ImageID:       "",
	})
	return ed
}

type testPublicIPsListener struct {
	data set.Set[net.IPAddress]
	t    *testing.T
}

func newTestPublicIPsListener(t *testing.T) *testPublicIPsListener {
	return &testPublicIPsListener{
		data: set.NewSet[net.IPAddress](),
		t:    t,
	}
}

func (p *testPublicIPsListener) String() string {
	return fmt.Sprintf("%s", maps.Keys(p.data))
}

func (p *testPublicIPsListener) OnUpdate(ips set.Set[net.IPAddress]) {
	p.data = ips
	p.t.Logf("Updatet public IPs to %s", p.String())
}
