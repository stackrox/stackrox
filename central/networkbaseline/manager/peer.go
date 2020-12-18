package manager

import (
	"sort"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
)

type peer struct {
	isIngress bool
	entity    networkgraph.Entity
	dstPort   uint32
	protocol  storage.L4Protocol
}

func convertPeersFromProto(protoPeers []*storage.NetworkBaselinePeer) map[peer]struct{} {
	out := make(map[peer]struct{}, len(protoPeers))
	for _, protoPeer := range protoPeers {
		entity := networkgraph.Entity{ID: protoPeer.GetEntity().GetInfo().GetId(), Type: protoPeer.GetEntity().GetInfo().GetType()}
		for _, props := range protoPeer.GetProperties() {
			out[peer{
				isIngress: props.GetIngress(),
				entity:    entity,
				dstPort:   props.GetPort(),
				protocol:  props.GetProtocol(),
			}] = struct{}{}
		}
	}
	return out
}

func convertPeersToProto(peerSet map[peer]struct{}) []*storage.NetworkBaselinePeer {
	if len(peerSet) == 0 {
		return nil
	}
	propertiesByEntity := make(map[networkgraph.Entity][]*storage.NetworkBaselineConnectionProperties)
	for peer := range peerSet {
		propertiesByEntity[peer.entity] = append(propertiesByEntity[peer.entity], &storage.NetworkBaselineConnectionProperties{
			Ingress:  peer.isIngress,
			Port:     peer.dstPort,
			Protocol: peer.protocol,
		})
	}
	out := make([]*storage.NetworkBaselinePeer, 0, len(propertiesByEntity))
	for entity, properties := range propertiesByEntity {
		sort.Slice(properties, func(i, j int) bool {
			if properties[i].Ingress != properties[j].Ingress {
				return properties[i].Ingress
			}
			if properties[i].Protocol != properties[j].Protocol {
				return properties[i].Protocol < properties[j].Protocol
			}
			return properties[i].Port < properties[j].Port
		})
		out = append(out, &storage.NetworkBaselinePeer{
			Entity: &storage.NetworkEntity{
				Info: &storage.NetworkEntityInfo{
					Type: entity.Type,
					Id:   entity.ID,
				},
			},
			Properties: properties,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].GetEntity().GetInfo().GetId() < out[j].GetEntity().GetInfo().GetId()
	})
	return out
}

func peerFromV1Peer(v1Peer *v1.NetworkBaselinePeer) peer {
	return peer{
		isIngress: v1Peer.GetIngress(),
		entity: networkgraph.Entity{
			Type: v1Peer.GetEntity().GetType(),
			ID:   v1Peer.GetEntity().GetId(),
		},
		dstPort:  v1Peer.GetPort(),
		protocol: v1Peer.GetProtocol(),
	}
}

// reversePeerView takes the passed peer, which is a peer with respect to the passed
// referenceDeploymentID, and returns the peer object that this deployment is from the
// _other_ deployment's point of view.
func reversePeerView(referenceDeploymentID string, p *peer) peer {
	return peer{
		isIngress: !p.isIngress,
		entity: networkgraph.Entity{
			Type: storage.NetworkEntityInfo_DEPLOYMENT,
			ID:   referenceDeploymentID,
		},
		dstPort:  p.dstPort,
		protocol: p.protocol,
	}
}
