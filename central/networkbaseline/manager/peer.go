package manager

import (
	"sort"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
)

var (
	entityTypeToName = map[storage.NetworkEntityInfo_Type]func(peer *storage.NetworkBaselinePeer) string{
		storage.NetworkEntityInfo_DEPLOYMENT: func(peer *storage.NetworkBaselinePeer) string {
			return peer.GetEntity().GetInfo().GetDeployment().GetName()
		},
		storage.NetworkEntityInfo_EXTERNAL_SOURCE: func(peer *storage.NetworkBaselinePeer) string {
			return peer.GetEntity().GetInfo().GetExternalSource().GetName()
		},
		storage.NetworkEntityInfo_INTERNET: func(peer *storage.NetworkBaselinePeer) string {
			return networkgraph.InternetExternalSourceName
		},
	}

	entityTypeToEntityInfoDesc = map[storage.NetworkEntityInfo_Type]func(name string, info *storage.NetworkEntityInfo){
		storage.NetworkEntityInfo_DEPLOYMENT: func(name string, info *storage.NetworkEntityInfo) {
			info.Desc = &storage.NetworkEntityInfo_Deployment_{
				Deployment: &storage.NetworkEntityInfo_Deployment{
					Name: name,
				},
			}
		},
		storage.NetworkEntityInfo_EXTERNAL_SOURCE: func(name string, info *storage.NetworkEntityInfo) {
			info.Desc = &storage.NetworkEntityInfo_ExternalSource_{
				ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
					Name: name,
				},
			}
		},
		storage.NetworkEntityInfo_INTERNET: func(name string, info *storage.NetworkEntityInfo) {
			// No-op.
		},
	}
)

type peer struct {
	isIngress bool
	entity    networkgraph.Entity
	name      string
	dstPort   uint32
	protocol  storage.L4Protocol
}

type entityWithName struct {
	networkgraph.Entity
	Name string
}

func convertPeersFromProto(protoPeers []*storage.NetworkBaselinePeer) (map[peer]struct{}, error) {
	out := make(map[peer]struct{}, len(protoPeers))
	for _, protoPeer := range protoPeers {
		entity := networkgraph.Entity{ID: protoPeer.GetEntity().GetInfo().GetId(), Type: protoPeer.GetEntity().GetInfo().GetType()}

		// Get name of entity based on type
		nameFn, ok := entityTypeToName[entity.Type]
		if !ok {
			// Not supported type
			return nil, errors.Errorf("unsupported entity type in network baseline: %q", entity.Type)
		}

		name := nameFn(protoPeer)
		for _, props := range protoPeer.GetProperties() {
			out[peer{
				isIngress: props.GetIngress(),
				entity:    entity,
				name:      name,
				dstPort:   props.GetPort(),
				protocol:  props.GetProtocol(),
			}] = struct{}{}
		}
	}
	return out, nil
}

func convertPeersToProto(peerSet map[peer]struct{}) ([]*storage.NetworkBaselinePeer, error) {
	if len(peerSet) == 0 {
		return nil, nil
	}
	propertiesByEntity := make(map[entityWithName][]*storage.NetworkBaselineConnectionProperties)
	for peer := range peerSet {
		entity := entityWithName{
			Entity: peer.entity,
			Name:   peer.name,
		}
		propertiesByEntity[entity] = append(propertiesByEntity[entity], &storage.NetworkBaselineConnectionProperties{
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

		// Get corresponding entity proto
		entityInfo := &storage.NetworkEntityInfo{
			Type: entity.Type,
			Id:   entity.ID,
		}
		infoDescFn, ok := entityTypeToEntityInfoDesc[entity.Type]
		if !ok {
			// Unsupported type
			return nil, errors.Errorf("unsupported entity type in network baseline: %q", entity.Type)
		}

		// Fill desc of info
		infoDescFn(entity.Name, entityInfo)
		out = append(out, &storage.NetworkBaselinePeer{
			Entity:     &storage.NetworkEntity{Info: entityInfo},
			Properties: properties,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].GetEntity().GetInfo().GetId() < out[j].GetEntity().GetInfo().GetId()
	})
	return out, nil
}

func peerFromV1Peer(v1Peer *v1.NetworkBaselineStatusPeer) peer {
	return peer{
		isIngress: v1Peer.GetIngress(),
		entity: networkgraph.Entity{
			Type: v1Peer.GetEntity().GetType(),
			ID:   v1Peer.GetEntity().GetId(),
		},
		name:     v1Peer.GetEntity().GetName(),
		dstPort:  v1Peer.GetPort(),
		protocol: v1Peer.GetProtocol(),
	}
}

// reversePeerView takes the passed peer, which is a peer with respect to the passed
// referenceDeploymentID, and returns the peer object that this deployment is from the
// _other_ deployment's point of view.
func reversePeerView(referenceDeploymentID, referenceDeploymentName string, p *peer) peer {
	return peer{
		isIngress: !p.isIngress,
		entity: networkgraph.Entity{
			Type: storage.NetworkEntityInfo_DEPLOYMENT,
			ID:   referenceDeploymentID,
		},
		name:     referenceDeploymentName,
		dstPort:  p.dstPort,
		protocol: p.protocol,
	}
}
