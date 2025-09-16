package networkgraph

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/networkgraph/externalsrcs"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	// EntityTypeToName is NetworkEntityInfo_Type to name function
	EntityTypeToName = map[storage.NetworkEntityInfo_Type]func(info *storage.NetworkEntityInfo) string{
		storage.NetworkEntityInfo_DEPLOYMENT: func(info *storage.NetworkEntityInfo) string {
			return info.GetDeployment().GetName()
		},
		storage.NetworkEntityInfo_EXTERNAL_SOURCE: func(info *storage.NetworkEntityInfo) string {
			return info.GetExternalSource().GetName()
		},
		storage.NetworkEntityInfo_INTERNET: func(info *storage.NetworkEntityInfo) string {
			return InternetExternalSourceName
		},
		storage.NetworkEntityInfo_INTERNAL_ENTITIES: func(info *storage.NetworkEntityInfo) string {
			return InternalEntitiesName
		},
	}

	ipv4InternetCIDR = "0.0.0.0/0"
	ipv6InternetCIDR = "::ffff:0:0/0"
)

// Entity represents a network entity in a form that is suitable for use as a map key.
type Entity struct {
	Type storage.NetworkEntityInfo_Type
	ID   string

	// Specific to ExternalSource entities
	ExternalEntityAddress net.IPNetwork
	Discovered            bool
}

// ToProto converts the Entity struct to a storage.NetworkEntityInfo proto.
func (e Entity) ToProto() *storage.NetworkEntityInfo {
	if e.Discovered && e.Type == storage.NetworkEntityInfo_EXTERNAL_SOURCE {
		return &storage.NetworkEntityInfo{
			Type: e.Type,
			Id:   e.ID,
			Desc: &storage.NetworkEntityInfo_ExternalSource_{
				ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
					Name:       e.ExternalEntityAddress.IP().String(),
					Default:    false,
					Discovered: true,
					Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
						Cidr: e.ExternalEntityAddress.String(),
					},
				},
			},
		}
	}
	return &storage.NetworkEntityInfo{
		Type: e.Type,
		Id:   e.ID,
	}
}

// EntityFromProto converts a storage.NetworkEntityInfo proto to an Entity struct.
func EntityFromProto(protoEnt *storage.NetworkEntityInfo) Entity {
	discovered := protoEnt.Type == storage.NetworkEntityInfo_EXTERNAL_SOURCE && protoEnt.GetExternalSource().GetDiscovered()
	extAddr := net.IPNetwork{}
	if discovered {
		extAddr = net.IPNetworkFromCIDR(protoEnt.GetExternalSource().GetCidr())
	}

	return Entity{
		Type:                  protoEnt.GetType(),
		ID:                    protoEnt.GetId(),
		Discovered:            discovered,
		ExternalEntityAddress: extAddr,
	}
}

// EntityForDeployment returns an Entity struct for the deployment with the given ID.
func EntityForDeployment(id string) Entity {
	return Entity{
		Type: storage.NetworkEntityInfo_DEPLOYMENT,
		ID:   id,
	}
}

// InternetEntity returns the de-facto INTERNET network entity to which all the connections to unidentified external sources are attributed to.
func InternetEntity() Entity {
	return Entity{
		ID:   InternetExternalSourceID,
		Type: storage.NetworkEntityInfo_INTERNET,
	}
}

// InternalEntities returns the internal-unknown network entity to which all the connections to unidentified internal sources are attributed to.
func InternalEntities() Entity {
	return Entity{
		ID:   InternalSourceID,
		Type: storage.NetworkEntityInfo_INTERNAL_ENTITIES,
	}
}

// DiscoveredExternalEntity returns an EXTERNAL_SOURCE entity refering to the provided network address.
// It is marked as "Discovered" to constrast with Entities defined by the user or the default ones.
func DiscoveredExternalEntity(address net.IPNetwork) Entity {
	id, err := externalsrcs.NewGlobalScopedScopedID(address.String())
	utils.Should(errors.Wrapf(err, "generating id for network %s", address.String()))

	return Entity{
		Type:                  storage.NetworkEntityInfo_EXTERNAL_SOURCE,
		ID:                    id.String(),
		ExternalEntityAddress: address,
		Discovered:            true,
	}
}

// DiscoveredExternalEntityClusterScoped returns an EXTERNAL_SOURCE entity refering to the provided
// cluster, and network address.
// It is marked as "Discovered" to constrast with Entities defined by the user or the default ones.
func DiscoveredExternalEntityClusterScoped(clusterId string, address net.IPNetwork) Entity {
	id, err := externalsrcs.NewClusterScopedID(clusterId, address.String())
	utils.Should(errors.Wrapf(err, "generating id for cluster/network %s/%s", clusterId, address.String()))

	return Entity{
		Type:                  storage.NetworkEntityInfo_EXTERNAL_SOURCE,
		ID:                    id.String(),
		ExternalEntityAddress: address,
		Discovered:            true,
	}
}

// InternetProtoWithDesc returns storage.NetworkEntityInfo proto object with Desc field filled in.
func InternetProtoWithDesc(family net.Family) *storage.NetworkEntityInfo {
	var cidr string
	if family == net.IPv4 {
		cidr = ipv4InternetCIDR
	} else if family == net.IPv6 {
		cidr = ipv6InternetCIDR
	} else {
		return nil
	}

	return &storage.NetworkEntityInfo{
		Id:   InternetExternalSourceID,
		Type: storage.NetworkEntityInfo_INTERNET,
		Desc: &storage.NetworkEntityInfo_ExternalSource_{
			ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
				Name: "External Entities",
				Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
					Cidr: cidr,
				},
			},
		},
	}
}
