package clusterentities

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/pkg/net"
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
	data map[net.IPAddress]struct{}
	t    *testing.T
}

func newTestPublicIPsListener(t *testing.T) *testPublicIPsListener {
	return &testPublicIPsListener{
		data: make(map[net.IPAddress]struct{}),
		t:    t,
	}
}

func (p *testPublicIPsListener) String() string {
	return fmt.Sprintf("%s", maps.Keys(p.data))
}

func (p *testPublicIPsListener) OnAdded(ip net.IPAddress) {
	p.data[ip] = struct{}{}
	p.t.Logf("Added new public IP %s", ip)
}

func (p *testPublicIPsListener) OnRemoved(ip net.IPAddress) {
	delete(p.data, ip)
	p.t.Logf("Deleted public IP %s", ip)
}
