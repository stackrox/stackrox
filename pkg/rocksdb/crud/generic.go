package generic

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/stackrox/pkg/db"
	"github.com/stackrox/stackrox/pkg/rocksdb"
)

// Deserializer is the function that takes in a []byte value and returns a proto message
type Deserializer func(v []byte) (proto.Message, error)

// AllocFunc returns an object of the type for the store
type AllocFunc func() proto.Message

// KeyFunc returns the key for the passed msg
type KeyFunc func(msg proto.Message) []byte

func deserializerFunc(alloc AllocFunc) Deserializer {
	return func(v []byte) (proto.Message, error) {
		t := alloc()
		if err := proto.Unmarshal(v, t); err != nil {
			return nil, err
		}
		return t, nil
	}
}

// NewCRUD returns a new Crud instance for the given bucket reference.
func NewCRUD(db *rocksdb.RocksDB, prefix []byte, keyFunc KeyFunc, alloc AllocFunc, trackIndex bool) db.Crud {
	return &crudImpl{
		db:        db,
		txnHelper: newTxnHelper(db, prefix, trackIndex),
		prefix:    prefix,

		keyFunc:         keyFunc,
		alloc:           alloc,
		deserializeFunc: deserializerFunc(alloc),
	}
}
