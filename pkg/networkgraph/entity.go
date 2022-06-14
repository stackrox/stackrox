package networkgraph

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/net"
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
	}

	ipv4InternetCIDR = "0.0.0.0/0"
	ipv6InternetCIDR = "::ffff:0:0/0"
)

// Entity represents a network entity in a form that is suitable for use as a map key.
type Entity struct {
	Type storage.NetworkEntityInfo_Type
	ID   string
}

// ToProto converts the Entity struct to a storage.NetworkEntityInfo proto.
func (e Entity) ToProto() *storage.NetworkEntityInfo {
	return &storage.NetworkEntityInfo{
		Type: e.Type,
		Id:   e.ID,
	}
}

// EntityFromProto converts a storage.NetworkEntityInfo proto to an Entity struct.
func EntityFromProto(protoEnt *storage.NetworkEntityInfo) Entity {
	return Entity{
		Type: protoEnt.GetType(),
		ID:   protoEnt.GetId(),
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
