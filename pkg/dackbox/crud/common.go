package crud

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/pkg/badgerhelper"
)

// ProtoAllocFunction allocates a proto object that we can deserialize data into.
type ProtoAllocFunction func() proto.Message

// ProtoKeyFunction provides the key for a given proto message.
type ProtoKeyFunction func(proto.Message) []byte

// PrefixKey prefixes the key returned from the input ProtoKeyFunction.
func PrefixKey(prefix []byte, function ProtoKeyFunction) ProtoKeyFunction {
	return func(msg proto.Message) []byte {
		return badgerhelper.GetBucketKey(prefix, function(msg))
	}
}

// ProtoMergeFunction merges a set of proto messages into a single result.
type ProtoMergeFunction func(base proto.Message, partials ...proto.Message) proto.Message

// ProtoSplitFunction splits an input object to a new object, and a set of partials
type ProtoSplitFunction func(base proto.Message) (proto.Message, []proto.Message)

// KeyMatchFunction returns whether a key should have some operation performed on it.
type KeyMatchFunction func([]byte) bool

// HasPrefix has a KeyMatchFunction that matches a key based on whether or not it has a prefix.
func HasPrefix(prefix []byte) KeyMatchFunction {
	return func(key []byte) bool {
		return badgerhelper.HasPrefix(prefix, key)
	}
}
