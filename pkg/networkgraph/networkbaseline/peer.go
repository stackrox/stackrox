package networkbaseline

import (
	"sort"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
)

var (
	// EntityTypeToEntityInfoDesc collects the functions to get names from corresponding network entity types
	EntityTypeToEntityInfoDesc = map[storage.NetworkEntityInfo_Type]func(name string, info *storage.NetworkEntityInfo, customProperties EntityProperties){
		storage.NetworkEntityInfo_DEPLOYMENT: func(name string, info *storage.NetworkEntityInfo, _ EntityProperties) {
			nd := &storage.NetworkEntityInfo_Deployment{}
			nd.SetName(name)
			info.SetDeployment(nd)
		},
		storage.NetworkEntityInfo_EXTERNAL_SOURCE: func(name string, info *storage.NetworkEntityInfo, custom EntityProperties) {
			ne := &storage.NetworkEntityInfo_ExternalSource{}
			ne.SetName(name)
			ne.SetCidr(custom.CIDRBlock)
			info.SetExternalSource(ne)
		},
		storage.NetworkEntityInfo_INTERNET: func(name string, info *storage.NetworkEntityInfo, _ EntityProperties) {
			// No-op.
		},
		storage.NetworkEntityInfo_INTERNAL_ENTITIES: func(name string, info *storage.NetworkEntityInfo, _ EntityProperties) {
			// No-op.
		},
	}

	// ValidBaselinePeerEntityTypes is a set of valid peer entity types that we currently support in network baseline
	ValidBaselinePeerEntityTypes = map[storage.NetworkEntityInfo_Type]struct{}{
		storage.NetworkEntityInfo_DEPLOYMENT:        {},
		storage.NetworkEntityInfo_EXTERNAL_SOURCE:   {},
		storage.NetworkEntityInfo_INTERNET:          {},
		storage.NetworkEntityInfo_INTERNAL_ENTITIES: {},
	}
)

// Peer is a in-memory representation of the network baseline peer
type Peer struct {
	IsIngress bool
	Entity    networkgraph.Entity
	Name      string
	DstPort   uint32
	Protocol  storage.L4Protocol

	// CidrBlock is specific to external source peers. This should only be filled when the underlying entity
	// is an external source. This is needed in order to differentiate baselines that are created to the same
	// provider/region but for different CIDR blocks. Check (https://github.com/stackrox/stackrox/pull/5194)
	CidrBlock string
}

type entityWithName struct {
	networkgraph.Entity
	Name string
}

// EntityProperties represents the properties of a peer entity for the baseline.
type EntityProperties struct {
	// CIDRBlock will only be filled if the peer entity is an External Source.
	CIDRBlock            string
	ConnectionProperties []*storage.NetworkBaselineConnectionProperties
}

// ConvertPeersFromProto converts proto NetworkBaselinePeer to its in memory representation
func ConvertPeersFromProto(protoPeers []*storage.NetworkBaselinePeer) (map[Peer]struct{}, error) {
	out := make(map[Peer]struct{}, len(protoPeers))
	for _, protoPeer := range protoPeers {
		entity := networkgraph.Entity{ID: protoPeer.GetEntity().GetInfo().GetId(), Type: protoPeer.GetEntity().GetInfo().GetType()}

		// Get name of entity based on type
		nameFn, ok := networkgraph.EntityTypeToName[entity.Type]
		if !ok {
			// Not supported type
			return nil, errors.Errorf("unsupported entity type in network baseline: %q", entity.Type)
		}

		// CIDR block is only set if the peer is of type External Source.
		var cidr string
		if entity.Type == storage.NetworkEntityInfo_EXTERNAL_SOURCE {
			cidr = protoPeer.GetEntity().GetInfo().GetExternalSource().GetCidr()
		}

		name := nameFn(protoPeer.GetEntity().GetInfo())
		for _, props := range protoPeer.GetProperties() {
			out[Peer{
				IsIngress: props.GetIngress(),
				Entity:    entity,
				Name:      name,
				DstPort:   props.GetPort(),
				Protocol:  props.GetProtocol(),
				CidrBlock: cidr,
			}] = struct{}{}
		}
	}
	return out, nil
}

// ConvertPeersToProto converts in-memory representation of network baseline peers to protos
func ConvertPeersToProto(peerSet map[Peer]struct{}) ([]*storage.NetworkBaselinePeer, error) {
	if len(peerSet) == 0 {
		return nil, nil
	}
	propertiesByEntity := make(map[entityWithName]EntityProperties)
	for peer := range peerSet {
		entity := entityWithName{
			Entity: peer.Entity,
			Name:   peer.Name,
		}

		if properties, ok := propertiesByEntity[entity]; ok {
			nbcp := &storage.NetworkBaselineConnectionProperties{}
			nbcp.SetIngress(peer.IsIngress)
			nbcp.SetPort(peer.DstPort)
			nbcp.SetProtocol(peer.Protocol)
			properties.ConnectionProperties = append(propertiesByEntity[entity].ConnectionProperties, nbcp)
			propertiesByEntity[entity] = properties
		} else {
			nbcp := &storage.NetworkBaselineConnectionProperties{}
			nbcp.SetIngress(peer.IsIngress)
			nbcp.SetPort(peer.DstPort)
			nbcp.SetProtocol(peer.Protocol)
			propertiesByEntity[entity] = EntityProperties{
				CIDRBlock: peer.CidrBlock,
				ConnectionProperties: []*storage.NetworkBaselineConnectionProperties{
					nbcp,
				},
			}
		}

	}

	out := make([]*storage.NetworkBaselinePeer, 0, len(propertiesByEntity))
	for entity, properties := range propertiesByEntity {
		connectionProperties := properties.ConnectionProperties
		sort.Slice(connectionProperties, func(i, j int) bool {
			if connectionProperties[i].GetIngress() != connectionProperties[j].GetIngress() {
				return connectionProperties[i].GetIngress()
			}
			if connectionProperties[i].GetProtocol() != connectionProperties[j].GetProtocol() {
				return connectionProperties[i].GetProtocol() < connectionProperties[j].GetProtocol()
			}
			return connectionProperties[i].GetPort() < connectionProperties[j].GetPort()
		})
		properties.ConnectionProperties = connectionProperties
		propertiesByEntity[entity] = properties

		// Get corresponding entity proto
		entityInfo := &storage.NetworkEntityInfo{}
		entityInfo.SetType(entity.Type)
		entityInfo.SetId(entity.ID)
		infoDescFn, ok := EntityTypeToEntityInfoDesc[entity.Type]
		if !ok {
			// Unsupported type
			return nil, errors.Errorf("unsupported entity type in network baseline: %q", entity.Type)
		}

		// Fill desc of info
		infoDescFn(entity.Name, entityInfo, EntityProperties{CIDRBlock: properties.CIDRBlock})
		ne := &storage.NetworkEntity{}
		ne.SetInfo(entityInfo)
		nbp := &storage.NetworkBaselinePeer{}
		nbp.SetEntity(ne)
		nbp.SetProperties(properties.ConnectionProperties)
		out = append(out, nbp)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].GetEntity().GetInfo().GetId() < out[j].GetEntity().GetInfo().GetId()
	})
	return out, nil
}

// PeerFromV1Peer converts peer within v1 request to in-memory representation form
func PeerFromV1Peer(v1Peer *v1.NetworkBaselineStatusPeer, peerName, cidrBlock string) Peer {
	return Peer{
		IsIngress: v1Peer.GetIngress(),
		Entity:    AnonymizeExternalDiscoveredPeer(v1Peer.GetEntity()),
		Name:      peerName,
		DstPort:   v1Peer.GetPort(),
		Protocol:  v1Peer.GetProtocol(),
		CidrBlock: cidrBlock,
	}
}

// PeerFromNetworkEntityInfo converts peer from storage.NetworkEntityInfo
func PeerFromNetworkEntityInfo(
	info *storage.NetworkEntityInfo,
	peerName string,
	dstPort uint32,
	protocol storage.L4Protocol,
	isIngressToBaselineEntity bool,
) Peer {
	entity := networkgraph.Entity{
		Type: info.GetType(),
		ID:   info.GetId(),
	}
	return PeerFromNetworkEntity(entity, peerName, dstPort, protocol, isIngressToBaselineEntity)
}

// PeerFromNetworkEntity converts peer from networkgraph.Entity
func PeerFromNetworkEntity(
	entity networkgraph.Entity,
	peerName string,
	dstPort uint32,
	protocol storage.L4Protocol,
	isIngressToBaselineEntity bool,
) Peer {
	return Peer{
		IsIngress: isIngressToBaselineEntity,
		Entity:    entity,
		Name:      peerName,
		DstPort:   dstPort,
		Protocol:  protocol,
	}
}

// ReversePeerView takes the passed peer, which is a peer with respect to the passed
// referenceDeploymentID, and returns the peer object that this deployment is from the
// _other_ deployment's point of view.
func ReversePeerView(referenceDeploymentID, referenceDeploymentName string, p *Peer) Peer {
	return Peer{
		IsIngress: !p.IsIngress,
		Entity: networkgraph.Entity{
			Type: storage.NetworkEntityInfo_DEPLOYMENT,
			ID:   referenceDeploymentID,
		},
		Name:     referenceDeploymentName,
		DstPort:  p.DstPort,
		Protocol: p.Protocol,
	}
}

// AnonymizeExternalDiscoveredPeer anonymizes a given baseline peer to the Internet
// if it a discovered external entity
func AnonymizeExternalDiscoveredPeer(peer *v1.NetworkBaselinePeerEntity) networkgraph.Entity {
	return AnonymizeExternalDiscoveredEntity(networkgraph.Entity{
		ID:         peer.GetId(),
		Type:       peer.GetType(),
		Discovered: peer.GetDiscovered(),
	})
}

// AnonymizeExternalDiscoveredEntity anonymizes a given graph entity to the Internet
// if it a discovered external entity
func AnonymizeExternalDiscoveredEntity(entity networkgraph.Entity) networkgraph.Entity {
	if entity.Type == storage.NetworkEntityInfo_EXTERNAL_SOURCE && entity.Discovered {
		return networkgraph.InternetEntity()
	}
	return entity
}
