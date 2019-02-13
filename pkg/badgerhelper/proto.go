package badgerhelper

import (
	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/proto"
)

// UnmarshalProtoValue unmarshals a protobuf-encoded value, avoiding extra copies.
func UnmarshalProtoValue(item *badger.Item, pb proto.Message) error {
	return item.Value(func(v []byte) error {
		return proto.Unmarshal(v, pb)
	})
}
