package m111tom112

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/m_105_to_m_106_group_id/binenc"
)

// NOTE: This file contains copied code originating from github.com/stackrox/rox/central/group/datastore/serialize.

func deserialize(key, value []byte) (*storage.Group, error) {
	props, err := deserializePropsKey(key)
	if err != nil {
		return nil, err
	}

	return &storage.Group{
		Props:    props,
		RoleName: string(value),
	}, nil
}

func deserializePropsKey(key []byte) (*storage.GroupProperties, error) {
	parts, err := binenc.DecodeBytesList(key)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode bytes list")
	}
	if len(parts) != 3 {
		return nil, errors.Errorf("decoded bytes list has %d elements, expected 3", len(parts))
	}

	if len(parts[0])+len(parts[1])+len(parts[2]) == 0 {
		return nil, nil
	}

	return &storage.GroupProperties{
		AuthProviderId: string(parts[0]),
		Key:            string(parts[1]),
		Value:          string(parts[2]),
	}, nil
}

func serialize(grp *storage.Group) ([]byte, []byte) {
	key := serializePropsKey(grp.GetProps())
	value := []byte(grp.GetRoleName())
	return key, value
}

func serializePropsKey(props *storage.GroupProperties) []byte {
	return binenc.EncodeBytesList([]byte(props.GetAuthProviderId()), []byte(props.GetKey()), []byte(props.GetValue()))
}
