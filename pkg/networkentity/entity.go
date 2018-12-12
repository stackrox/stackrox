package networkentity

import (
	"github.com/stackrox/rox/generated/storage"
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

// FromProto converts a storage.NetworkEntityInfo proto to an Entity struct.
func FromProto(protoEnt *storage.NetworkEntityInfo) Entity {
	return Entity{
		Type: protoEnt.GetType(),
		ID:   protoEnt.GetId(),
	}
}

// ForDeployment returns an Entity struct for the deployment with the given ID.
func ForDeployment(id string) Entity {
	return Entity{
		Type: storage.NetworkEntityInfo_DEPLOYMENT,
		ID:   id,
	}
}
