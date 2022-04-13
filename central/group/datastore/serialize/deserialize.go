package serialize

import (
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/binenc"
)

// DeserializePropsKey deserializes a key serialized via `PropsKey`.
func DeserializePropsKey(key []byte) (*storage.GroupProperties, error) {
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
