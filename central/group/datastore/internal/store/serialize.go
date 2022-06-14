package store

import (
	serializePkg "github.com/stackrox/stackrox/central/group/datastore/serialize"
	"github.com/stackrox/stackrox/generated/storage"
)

// Serialization
////////////////

func serialize(group *storage.Group) ([]byte, []byte) {
	return serializePkg.PropsKey(group.GetProps()), serializeValue(group)
}

func serializeValue(group *storage.Group) []byte {
	return []byte(group.GetRoleName())
}

// Deserialization
////////////////

func deserialize(key, value []byte) (*storage.Group, error) {
	props, err := serializePkg.DeserializePropsKey(key)
	if err != nil {
		return nil, err
	}
	return &storage.Group{
		Props:    props,
		RoleName: string(value),
	}, nil
}
