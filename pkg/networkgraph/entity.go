package networkgraph

import (
	"github.com/stackrox/rox/generated/storage"
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
