package generic

import (
	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/utils"
)

// Deserializer is the function that takes in a []byte value and returns a proto message
type Deserializer func(v []byte) (proto.Message, error)

// allocFunc returns an object of the type for the store
type allocFunc func() proto.Message

type keyFunc func(msg proto.Message) []byte

// Crud provides a simple crud layer on top of Badger DB supporting proto messages
type Crud interface {
	Count() (int, error)

	Exists(id string) (bool, error)

	Read(id string) (proto.Message, bool, error)
	ReadPartial(id string) (proto.Message, bool, error)

	ReadBatch(ids []string) (msgs []proto.Message, indices []int, err error)
	ReadBatchPartial(ids []string) (msgs []proto.Message, indices []int, err error)

	ReadAll() (msgs []proto.Message, err error)
	ReadAllPartial() (msgs []proto.Message, err error)

	Upsert(kv proto.Message) error
	UpsertBatch(msgs []proto.Message) error

	Delete(id string) error
	DeleteBatch(ids []string) error

	AddKeysToIndex(tx badgerhelper.TxWrapper, keys ...[]byte) error
	AckKeysIndexed(keys ...string) error
	GetKeysToIndex() ([]string, error)

	GetKeys() ([]string, error)
}

func deserializerFunc(alloc allocFunc) Deserializer {
	return func(v []byte) (proto.Message, error) {
		t := alloc()
		if err := proto.Unmarshal(v, t); err != nil {
			return nil, err
		}
		return t, nil
	}
}

// NewCRUD returns a new Crud instance for the given bucket reference.
func NewCRUD(db *badger.DB, prefix []byte, keyFunc keyFunc, alloc allocFunc) Crud {
	helper, err := badgerhelper.NewTxnHelper(db, prefix)
	utils.Must(err)

	return &crudImpl{
		db:        db,
		TxnHelper: helper,
		prefix:    prefix,

		keyFunc:         keyFunc,
		alloc:           alloc,
		deserializeFunc: deserializerFunc(alloc),
		hasPartial:      false,
	}
}

// NewCRUDWithPartial creates a CRUD store with a partial bucket as well
func NewCRUDWithPartial(db *badger.DB, prefix []byte, keyFunc keyFunc, alloc allocFunc,
	partialPrefix []byte, partialAlloc allocFunc, partialConverter partialConvert) Crud {
	helper, err := badgerhelper.NewTxnHelper(db, prefix)
	utils.Must(err)

	return &crudImpl{
		db:              db,
		TxnHelper:       helper,
		prefix:          prefix,
		prefixString:    string(prefix),
		keyFunc:         keyFunc,
		alloc:           alloc,
		deserializeFunc: deserializerFunc(alloc),

		hasPartial:             true,
		partialPrefix:          partialPrefix,
		partialAlloc:           partialAlloc,
		partialConverter:       partialConverter,
		partialDeserializeFunc: deserializerFunc(partialAlloc),
	}
}
