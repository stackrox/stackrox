package crud

import (
	"github.com/stackrox/rox/pkg/dbhelper"
	"github.com/stackrox/rox/pkg/protocompat"
)

// ProtoAllocFunction allocates a proto object that we can deserialize data into.
type ProtoAllocFunction func() protocompat.Message

// ProtoKeyFunction provides the key for a given proto message.
type ProtoKeyFunction func(protocompat.Message) []byte

// PrefixKey prefixes the key returned from the input ProtoKeyFunction.
func PrefixKey(prefix []byte, function ProtoKeyFunction) ProtoKeyFunction {
	return func(msg protocompat.Message) []byte {
		return dbhelper.GetBucketKey(prefix, function(msg))
	}
}
