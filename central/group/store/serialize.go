package store

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/generated/api/v1"
)

// Serialization
////////////////

func serialize(group *v1.Group) ([]byte, []byte) {
	return serializeKey(group.GetProps()), serializeValue(group)
}

func serializeKey(props *v1.GroupProperties) []byte {
	return serializeKeyProps(props.GetAuthProviderId(), props.GetKey(), props.GetValue())
}

func serializeKeyProps(authProviderID, key, value string) []byte {
	return []byte(fmt.Sprintf("%s:%s:%s", authProviderID, key, value))
}

func serializeValue(group *v1.Group) []byte {
	return []byte(group.GetRoleName())
}

// Deserialization
////////////////

func deserialize(key, value []byte) (*v1.Group, error) {
	props, err := deserializeKey(key)
	if err != nil {
		return nil, err
	}
	return &v1.Group{
		Props:    props,
		RoleName: string(value),
	}, nil
}

func deserializeKey(key []byte) (*v1.GroupProperties, error) {
	str := string(key)
	props := strings.Split(str, ":")
	if len(props) != 3 {
		return nil, fmt.Errorf("unable to deserialize key: %s", str)
	}
	// If no values (which is totally ok), then just return nil.
	if len(props[0]) == 0 && len(props[1]) == 0 && len(props[2]) == 0 {
		return nil, nil
	}
	return &v1.GroupProperties{
		AuthProviderId: props[0],
		Key:            props[1],
		Value:          props[2],
	}, nil
}
