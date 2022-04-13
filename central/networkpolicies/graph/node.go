package graph

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/protoconv/k8s"
)

// namedPort identifies a port that is referenced by name.
type namedPort struct {
	l4Proto  storage.Protocol
	portName string
}

type node struct {
	deployment *storage.Deployment
	extSrc     *storage.NetworkEntityInfo

	isIngressIsolated bool
	isEgressIsolated  bool

	internetAccess bool

	applyingPoliciesIDs []string

	namedContainerPorts map[namedPort]int32

	ingressEdges  map[*node]*edge
	egressEdges   map[*node]*edge
	adjacentNodes map[*node]struct{}
}

func newDeploymentNode(d *storage.Deployment) *node {
	n := &node{
		deployment: d,
	}
	n.initNamedPorts()

	return n
}

func newExternalSrcNode(extSrc *storage.NetworkEntityInfo) *node {
	n := &node{
		extSrc: extSrc,
	}
	return n
}

func (d *node) toEntityProto() *storage.NetworkEntityInfo {
	if d.extSrc == nil && d.deployment == nil {
		return nil
	}

	if d.extSrc != nil {
		return d.extSrc
	}

	return &storage.NetworkEntityInfo{
		Type: storage.NetworkEntityInfo_DEPLOYMENT,
		Id:   d.deployment.GetId(),
		Desc: &storage.NetworkEntityInfo_Deployment_{
			Deployment: &storage.NetworkEntityInfo_Deployment{
				Name:      d.deployment.GetName(),
				Namespace: d.deployment.GetNamespace(),
			},
		},
	}
}

func (d *node) initNamedPorts() {
	if d.deployment == nil {
		return
	}

	d.namedContainerPorts = make(map[namedPort]int32)
	for _, portConfig := range d.deployment.GetPorts() {
		name := portConfig.GetName()
		if name == "" {
			continue
		}
		npRef := namedPort{
			l4Proto:  k8s.ProtoNameToStorageProtocol(portConfig.GetProtocol()),
			portName: name,
		}
		if npRef.l4Proto == storage.Protocol_UNSET_PROTOCOL {
			continue
		}
		d.namedContainerPorts[npRef] = portConfig.GetContainerPort()
	}
}

func (d *node) portByName(l4Proto storage.Protocol, portName string) int32 {
	return d.namedContainerPorts[namedPort{l4Proto: l4Proto, portName: portName}]
}

func (d *node) resolvePorts(ports []*storage.NetworkPolicyPort) portDescs {
	if len(ports) == 0 {
		return portDescs{{}} // all ports, all protocols
	}
	pds := make([]portDesc, 0, len(ports))
	for _, p := range ports {
		l4Proto := p.GetProtocol()
		if l4Proto == storage.Protocol_UNSET_PROTOCOL {
			l4Proto = storage.Protocol_TCP_PROTOCOL
		}

		var portNum int32
		switch r := p.GetPortRef().(type) {
		case *storage.NetworkPolicyPort_Port:
			portNum = r.Port
		case *storage.NetworkPolicyPort_PortName:
			portNum = d.portByName(l4Proto, r.PortName)
		}

		if portNum == 0 && p.GetPortRef() != nil {
			// Invalid port name or number.
			// Note that a network policy that matches all ports will always have a
			// nil PortRef, even if it was written before we moved the `port` field inside
			// of the oneof. This is because proto3 ensures that zero-valued fields outside of
			// oneof blocks are never written to the wire.
			continue
		}

		pds = append(pds, portDesc{l4proto: l4Proto, port: portNum})
	}

	return pds
}

type edge struct {
	src, tgt *node

	ports portDescs
}

func (e *edge) getPorts() portDescs {
	if e == nil {
		return nil
	}
	return e.ports
}
