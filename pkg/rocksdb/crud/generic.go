package generic

import (
	"github.com/stackrox/rox/pkg/db"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/rocksdb"
)

// Deserializer is the function that takes in a []byte value and returns a proto message
type Deserializer func(v []byte) (protocompat.Message, error)

// AllocFunc returns an object of the type for the store
type AllocFunc func() protocompat.Message

// KeyFunc returns the key for the passed msg
type KeyFunc func(msg protocompat.Message) []byte

func deserializerFunc(alloc AllocFunc) Deserializer {
	return func(v []byte) (protocompat.Message, error) {
		t := alloc()
		if err := protocompat.Unmarshal(v, t); err != nil {
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
