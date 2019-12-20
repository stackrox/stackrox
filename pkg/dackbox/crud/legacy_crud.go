package crud

import (
	"github.com/gogo/protobuf/proto"
	generic "github.com/stackrox/rox/pkg/badgerhelper/crud"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/utils"
)

// allocFunc returns an object of the type for the store
type allocFunc func() proto.Message

type keyFunc func(msg proto.Message) []byte

type partialConvert func(msg proto.Message) proto.Message

// NewCRUD returns a new Crud instance for the given bucket reference.
func NewCRUD(duckBox *dackbox.DackBox, prefix []byte, keyFunc keyFunc, alloc allocFunc) generic.Crud {
	helper, err := NewTxnCounter(duckBox, prefix)
	utils.Must(err)
	return &legacyCrudImpl{
		counter:  helper,
		duckBox:  duckBox,
		prefix:   prefix,
		reader:   NewReader(WithAllocFunction(ProtoAllocFunction(alloc))),
		upserter: NewUpserter(WithKeyFunction(PrefixKey(prefix, ProtoKeyFunction(keyFunc)))),
		deleter:  NewDeleter(GCAllChildren()),
	}
}

// NewCRUDWithPartial creates a CRUD store with a partial bucket as well
func NewCRUDWithPartial(duckBox *dackbox.DackBox, prefix []byte, keyFunc keyFunc, alloc allocFunc,
	partialPrefix []byte, partialAlloc allocFunc, partialConverter partialConvert) generic.Crud {
	helper, err := NewTxnCounter(duckBox, prefix)
	utils.Must(err)

	splitFunc := func(msg proto.Message) (proto.Message, []proto.Message) {
		return msg, []proto.Message{partialConverter(msg)}
	}

	return &legacyCrudImpl{
		counter:    helper,
		duckBox:    duckBox,
		prefix:     prefix,
		listPrefix: partialPrefix,
		reader:     NewReader(WithAllocFunction(ProtoAllocFunction(alloc))),
		listReader: NewReader(WithAllocFunction(ProtoAllocFunction(partialAlloc))),
		upserter: NewUpserter(
			WithKeyFunction(PrefixKey(prefix, ProtoKeyFunction(keyFunc))),
			WithPartialUpserter(
				NewPartialUpserter(
					WithSplitFunc(splitFunc),
					WithUpserter(
						NewUpserter(
							WithKeyFunction(PrefixKey(partialPrefix, ProtoKeyFunction(keyFunc))),
						),
					),
				),
			),
		),
		deleter: NewDeleter(GCAllChildren()),
	}
}
