package networkentity

import "github.com/stackrox/rox/generated/api/v1"

// Entity represents a network entity in a form that is suitable for use as a map key.
type Entity struct {
	Type v1.NetworkEntityInfo_Type
	ID   string
}

// ToProto converts the Entity struct to a v1.NetworkEntityInfo proto.
func (e Entity) ToProto() *v1.NetworkEntityInfo {
	return &v1.NetworkEntityInfo{
		Type: e.Type,
		Id:   e.ID,
	}
}

// FromProto converts a v1.NetworkEntityInfo proto to an Entity struct.
func FromProto(protoEnt *v1.NetworkEntityInfo) Entity {
	return Entity{
		Type: protoEnt.GetType(),
		ID:   protoEnt.GetId(),
	}
}

// ForDeployment returns an Entity struct for the deployment with the given ID.
func ForDeployment(id string) Entity {
	return Entity{
		Type: v1.NetworkEntityInfo_DEPLOYMENT,
		ID:   id,
	}
}
