package crud

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/pkg/dackbox"
)

// allocFunc returns an object of the type for the store
type allocFunc func() proto.Message

type keyFunc func(msg proto.Message) []byte

type partialConvert func(msg proto.Message) proto.Message

func newCrud(duckBox *dackbox.DackBox, prefix []byte, keyFunc keyFunc, alloc allocFunc) *legacyCrudImpl {
	return &legacyCrudImpl{
		duckBox:  duckBox,
		prefix:   prefix,
		reader:   NewReader(WithAllocFunction(ProtoAllocFunction(alloc))),
		upserter: NewUpserter(WithKeyFunction(PrefixKey(prefix, ProtoKeyFunction(keyFunc)))),
		deleter:  NewDeleter(),
	}
}

func newCrudWithPartial(duckBox *dackbox.DackBox, prefix []byte, keyFunc keyFunc, alloc allocFunc,
	partialPrefix []byte, partialAlloc allocFunc, partialConverter partialConvert) *legacyCrudImpl {

	splitFunc := func(msg proto.Message) (proto.Message, []proto.Message) {
		return msg, []proto.Message{partialConverter(msg)}
	}

	return &legacyCrudImpl{
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
		deleter: NewDeleter(
			WithPartialDeleter(
				NewPartialDeleter(
					WithDeleter(
						NewDeleter(),
					),
				),
			),
		),
	}
}
