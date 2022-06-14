package crud

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/pkg/dbhelper"
)

// ProtoAllocFunction allocates a proto object that we can deserialize data into.
type ProtoAllocFunction func() proto.Message

// ProtoKeyFunction provides the key for a given proto message.
type ProtoKeyFunction func(proto.Message) []byte

// PrefixKey prefixes the key returned from the input ProtoKeyFunction.
func PrefixKey(prefix []byte, function ProtoKeyFunction) ProtoKeyFunction {
	return func(msg proto.Message) []byte {
		return dbhelper.GetBucketKey(prefix, function(msg))
	}
}
